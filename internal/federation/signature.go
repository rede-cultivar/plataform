package federation

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// signatureAlgorithm é o único algoritmo aceito nesta implementação.
	signatureAlgorithm = "rsa-sha256"
	// signatureClockSkew limita a diferença aceita entre as datas das instâncias.
	signatureClockSkew = 5 * time.Minute
	// maxRemoteDocumentBytes limita a leitura de documentos remotos.
	maxRemoteDocumentBytes = 1 << 20
)

// Erros de assinatura e de documentos remotos.
var (
	ErrMissingSignature = errors.New("assinatura HTTP ausente")
	ErrInvalidSignature = errors.New("assinatura HTTP inválida")
	ErrInvalidActivity  = errors.New("atividade federada inválida")
)

// SignatureParams contém os campos do cabeçalho Signature.
type SignatureParams struct {
	KeyID     string
	Algorithm string
	Headers   []string
	Signature string
}

// ParseSignatureHeader interpreta um cabeçalho Signature no formato
// keyId="...",algorithm="...",headers="...",signature="...".
func ParseSignatureHeader(value string) (*SignatureParams, error) {
	params := &SignatureParams{}
	fields := make([]string, 0, 4)

	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		key, raw, ok := strings.Cut(part, "=")
		if !ok {
			return nil, fmt.Errorf("%w: campo sem valor", ErrInvalidSignature)
		}
	fields = append(fields, strings.TrimSpace(key))

		raw = strings.Trim(strings.TrimSpace(raw), `"`)
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "keyid":
			params.KeyID = raw
		case "algorithm":
			params.Algorithm = raw
		case "headers":
			params.Headers = strings.Fields(strings.ToLower(raw))
		case "signature":
			params.Signature = raw
		}
	}

	if params.KeyID == "" || params.Signature == "" {
		return nil, fmt.Errorf("%w: keyId ou signature ausente (campos recebidos: %s)", ErrInvalidSignature, strings.Join(fields, ", "))
	}
	if params.Algorithm == "" {
		params.Algorithm = signatureAlgorithm
	}
	if params.Algorithm != signatureAlgorithm {
		return nil, fmt.Errorf("%w: algoritmo %q não suportado", ErrInvalidSignature, params.Algorithm)
	}
	if len(params.Headers) == 0 {
		params.Headers = []string{"(request-target)", "host", "date"}
	}

	return params, nil
}

// SignRequest assina uma requisição HTTP de saída com a chave do Actor.
//
// A assinatura cobre o método, o caminho, o host, a data, o resumo do corpo e o
// tipo de conteúdo, conforme a convenção usada pelo Mastodon.
func SignRequest(req *http.Request, body []byte, keyID string, key *rsa.PrivateKey) error {
	if req.Host == "" {
		req.Host = req.URL.Host
	}
	if req.Header.Get("Date") == "" {
		req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", ActivityContentType)
	}
	req.Header.Set("Digest", digestValue(body))

	headers := []string{"(request-target)", "host", "date", "digest", "content-type"}
	signingString := buildSigningString(req, headers)

	hashed := sha256.Sum256([]byte(signingString))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hashed[:])
	if err != nil {
		return fmt.Errorf("assinar requisição: %w", err)
	}

	req.Header.Set("Signature", fmt.Sprintf(
		`keyId="%s",algorithm="%s",headers="%s",signature="%s"`,
		keyID,
		signatureAlgorithm,
		strings.Join(headers, " "),
		base64.StdEncoding.EncodeToString(signature),
	))

	return nil
}

// Verifier valida assinaturas HTTP de atividades recebidas.
type Verifier struct {
	client    *http.Client
	now       func() time.Time
	clockSkew time.Duration
}

// NewVerifier cria um verificador que busca chaves públicas pela rede.
func NewVerifier(client *http.Client) *Verifier {
	if client == nil {
		client = NewHTTPClient(false)
	}
	return &Verifier{
		client:    client,
		now:       time.Now,
		clockSkew: signatureClockSkew,
	}
}

// VerifyRequest valida a assinatura, o resumo e a data da requisição.
//
// O corpo é recebido separadamente porque precisa ser lido antes da
// verificação e reutilizado pelo processamento da atividade.
func (v *Verifier) VerifyRequest(ctx context.Context, req *http.Request, body []byte) (*RemoteActor, error) {
	raw := req.Header.Get("Signature")
	if strings.TrimSpace(raw) == "" {
		return nil, ErrMissingSignature
	}

	params, err := ParseSignatureHeader(raw)
	if err != nil {
		return nil, err
	}

	if err := v.validateDate(req.Header.Get("Date")); err != nil {
		return nil, err
	}

	if req.Method == http.MethodPost {
		if !validDigest(req.Header.Get("Digest"), body) {
			return nil, fmt.Errorf("%w: digest do corpo não confere", ErrInvalidSignature)
		}
	}

	for _, header := range params.Headers {
		if header == "(request-target)" {
			continue
		}
		if strings.TrimSpace(signedHeaderValue(req, header)) == "" {
			return nil, fmt.Errorf("%w: cabeçalho %q ausente", ErrInvalidSignature, header)
		}
	}

	actor, key, err := v.fetchKey(ctx, params.KeyID)
	if err != nil {
		return nil, err
	}

	signature, err := base64.StdEncoding.DecodeString(params.Signature)
	if err != nil {
		return nil, fmt.Errorf("%w: assinatura não é base64", ErrInvalidSignature)
	}

	signingString := buildSigningString(req, params.Headers)
	hashed := sha256.Sum256([]byte(signingString))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, hashed[:], signature); err != nil {
		return nil, fmt.Errorf("%w: verificação RSA falhou", ErrInvalidSignature)
	}

	return actor, nil
}

// FetchActor busca e interpreta o documento Actor de outra instância.
func (v *Verifier) FetchActor(ctx context.Context, actorURL string) (*RemoteActor, error) {
	parsed, err := url.Parse(actorURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("%w: endereço de Actor inválido", ErrInvalidActivity)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, actorURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", ActivityContentType+`, application/ld+json; profile="https://www.w3.org/ns/activitystreams"`)

	resp, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("buscar Actor remoto: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("buscar Actor remoto: status %d", resp.StatusCode)
	}

	var actor RemoteActor
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxRemoteDocumentBytes)).Decode(&actor); err != nil {
		return nil, fmt.Errorf("interpretar Actor remoto: %w", err)
	}

	if actor.ID == "" || actor.Inbox == "" || actor.PublicKey.PublicKeyPEM == "" {
		return nil, fmt.Errorf("%w: Actor remoto incompleto", ErrInvalidActivity)
	}
	if !sameURL(actor.ID, actorURL) {
		return nil, fmt.Errorf("%w: identificador do Actor não confere", ErrInvalidActivity)
	}

	return &actor, nil
}

func (v *Verifier) validateDate(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%w: data ausente", ErrInvalidSignature)
	}

	date, err := http.ParseTime(value)
	if err != nil {
		return fmt.Errorf("%w: data inválida", ErrInvalidSignature)
	}

	difference := v.now().Sub(date)
	if difference > v.clockSkew || difference < -v.clockSkew {
		return fmt.Errorf("%w: data fora da janela aceita", ErrInvalidSignature)
	}

	return nil
}

func (v *Verifier) fetchKey(ctx context.Context, keyID string) (*RemoteActor, *rsa.PublicKey, error) {
	parsed, err := url.Parse(keyID)
	if err != nil || parsed.Fragment == "" {
		return nil, nil, fmt.Errorf("%w: keyId inválido", ErrInvalidSignature)
	}

	actorURL := *parsed
	actorURL.Fragment = ""

	actor, err := v.FetchActor(ctx, actorURL.String())
	if err != nil {
		return nil, nil, err
	}

	if actor.PublicKey.ID != "" && actor.PublicKey.ID != keyID {
		return nil, nil, fmt.Errorf("%w: chave não pertence ao keyId informado", ErrInvalidSignature)
	}
	if actor.PublicKey.Owner != "" && !sameURL(actor.PublicKey.Owner, actor.ID) {
		return nil, nil, fmt.Errorf("%w: chave não pertence ao Actor", ErrInvalidSignature)
	}

	key, err := ParsePublicKey(actor.PublicKey.PublicKeyPEM)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: chave pública inválida", ErrInvalidSignature)
	}

	return actor, key, nil
}

// buildSigningString monta as linhas assinadas na ordem declarada no cabeçalho.
func buildSigningString(req *http.Request, headers []string) string {
	lines := make([]string, 0, len(headers))

	for _, header := range headers {
		switch strings.ToLower(header) {
		case "(request-target)":
			lines = append(lines, "(request-target): "+strings.ToLower(req.Method)+" "+req.URL.RequestURI())
		default:
			lines = append(lines, strings.ToLower(header)+": "+signedHeaderValue(req, header))
		}
	}

	return strings.Join(lines, "\n")
}

func signedHeaderValue(req *http.Request, header string) string {
	if strings.EqualFold(header, "host") {
		return req.Host
	}
	return req.Header.Get(header)
}

func digestValue(body []byte) string {
	sum := sha256.Sum256(body)
	return "SHA-256=" + base64.StdEncoding.EncodeToString(sum[:])
}

func validDigest(header string, body []byte) bool {
	expected := digestValue(body)
	for _, part := range strings.Split(header, ",") {
		value := strings.TrimSpace(part)
		if subtle.ConstantTimeCompare([]byte(value), []byte(expected)) == 1 {
			return true
		}
	}
	return false
}

func sameURL(left, right string) bool {
	leftURL, err := url.Parse(left)
	if err != nil {
		return false
	}
	rightURL, err := url.Parse(right)
	if err != nil {
		return false
	}

	return strings.EqualFold(leftURL.Scheme, rightURL.Scheme) &&
		strings.EqualFold(leftURL.Host, rightURL.Host) &&
		strings.TrimSuffix(leftURL.Path, "/") == strings.TrimSuffix(rightURL.Path, "/")
}
