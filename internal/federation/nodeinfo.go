package federation

import "github.com/rede-cultivar/plataform/internal/identity"

const (
	// ActivityContentType é o tipo de mídia dos documentos ActivityPub.
	ActivityContentType = "application/activity+json; charset=utf-8"
	// JRDContentType é o tipo de mídia dos documentos WebFinger e de descoberta.
	JRDContentType = "application/jrd+json; charset=utf-8"
	// NodeInfoContentType é o tipo de mídia do documento NodeInfo 2.0.
	NodeInfoContentType = `application/json; profile="http://nodeinfo.diaspora.software/ns/schema/2.0#"`

	// NodeInfoVersion é a versão do esquema NodeInfo publicada.
	NodeInfoVersion = "2.0"
	// NodeInfoSchemaRel identifica o esquema NodeInfo 2.0.
	NodeInfoSchemaRel = "http://nodeinfo.diaspora.software/ns/schema/2.0"
)

// NodeInfoLink representa um link do documento de descoberta NodeInfo.
type NodeInfoLink struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

// NodeInfoIndex representa o documento de descoberta NodeInfo.
type NodeInfoIndex struct {
	Links []NodeInfoLink `json:"links"`
}

// NodeInfoSoftware descreve o software da instância.
type NodeInfoSoftware struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// NodeInfoServices lista os serviços de entrada e saída suportados.
type NodeInfoServices struct {
	Inbound  []string `json:"inbound"`
	Outbound []string `json:"outbound"`
}

// NodeInfoUsers reúne as contagens de pessoas.
type NodeInfoUsers struct {
	Total          int `json:"total"`
	ActiveMonth    int `json:"activeMonth"`
	ActiveHalfyear int `json:"activeHalfyear"`
}

// NodeInfoUsage reúne as estatísticas de uso da instância.
type NodeInfoUsage struct {
	Users      NodeInfoUsers `json:"users"`
	LocalPosts int           `json:"localPosts"`
}

// NodeInfoMetadata reúne informações complementares da instância.
type NodeInfoMetadata struct {
	NodeName string `json:"nodeName,omitempty"`
}

// NodeInfo representa o documento NodeInfo 2.0 da instância.
type NodeInfo struct {
	Version           string           `json:"version"`
	Software          NodeInfoSoftware `json:"software"`
	Protocols         []string         `json:"protocols"`
	Services          NodeInfoServices `json:"services"`
	OpenRegistrations bool             `json:"openRegistrations"`
	Usage             NodeInfoUsage    `json:"usage"`
	Metadata          NodeInfoMetadata `json:"metadata"`
}

// NewNodeInfoIndex monta o documento de descoberta do NodeInfo.
func NewNodeInfoIndex(urls URLs) NodeInfoIndex {
	return NodeInfoIndex{
		Links: []NodeInfoLink{
			{
				Rel:  NodeInfoSchemaRel,
				Href: urls.Base() + "/nodeinfo/2.0",
			},
		},
	}
}

// NewNodeInfo monta o documento NodeInfo 2.0 da instância.
func NewNodeInfo(instanceName, softwareVersion string, stats identity.Stats, registrationsOpen ...bool) NodeInfo {
	open := true
	if len(registrationsOpen) > 0 {
		open = registrationsOpen[0]
	}

	return NodeInfo{
		Version: NodeInfoVersion,
		Software: NodeInfoSoftware{
			Name:    "cultivar",
			Version: softwareVersion,
		},
		Protocols: []string{"activitypub"},
		Services: NodeInfoServices{
			Inbound:  []string{},
			Outbound: []string{},
		},
		OpenRegistrations: open,
		Usage: NodeInfoUsage{
			Users: NodeInfoUsers{
				Total:          stats.TotalAccounts,
				ActiveMonth:    stats.ActiveMonth,
				ActiveHalfyear: stats.ActiveHalfyear,
			},
			LocalPosts: 0,
		},
		Metadata: NodeInfoMetadata{
			NodeName: instanceName,
		},
	}
}
