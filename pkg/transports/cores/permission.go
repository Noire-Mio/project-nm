package cores

import "fmt"

var (
	CreateAction = NewPermissionNode("create", Language{"en": "create", "zh": "新增"})
	ReadAction   = NewPermissionNode("read", Language{"en": "read", "zh": "讀取"})
	UpdateAction = NewPermissionNode("update", Language{"en": "update", "zh": "更改"})
	DeleteAction = NewPermissionNode("delete", Language{"en": "delete", "zh": "刪除"})
)

type Language map[string]string

type Permission struct {
	Name      string   `json:"name"`
	Domain    string   `json:"domain"`
	Languages Language `json:"languages"`
}

func NewPermission(domain *PermissionNode, nodes ...*PermissionNode) *Permission {
	name := domain.Name
	languages := make(Language)
	for key, text := range domain.Languages {
		languages[key] = text
	}
	for _, node := range nodes {
		if len(domain.Languages) != len(node.Languages) {
			panic("some language is not fully supported")
		}
		name = fmt.Sprintf("%s.%s", name, node.Name)
		for key, text := range languages {
			languages[key] = fmt.Sprintf("%s.%s", text, node.Languages[key])
		}
	}
	return &Permission{Name: name, Domain: domain.Name, Languages: languages}
}

type PermissionNode struct {
	Name      string   `json:"name"`
	Languages Language `json:"languages"`
}

func NewPermissionNode(name string, languages map[string]string) *PermissionNode {
	if len(languages) == 0 {
		languages = make(map[string]string)
	}
	return &PermissionNode{Name: name, Languages: languages}
}

type PermissionMap map[string]*Permission

func (pm PermissionMap) Permissions() []Permission {
	ps := make([]Permission, len(pm))
	i := 0
	for _, p := range pm {
		ps[i] = *p
		i++
	}
	return ps
}

func (pm PermissionMap) AddPermission(domain, resource *PermissionNode, actions ...*PermissionNode) {
	for _, action := range actions {
		p := NewPermission(domain, resource, action)
		pm[p.Name] = p
	}
}
