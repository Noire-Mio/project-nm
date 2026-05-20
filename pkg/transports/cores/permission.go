package cores

type Permission struct {
	Name string
}

const (
	ActionCreate = "create"
	ActionRead   = "read"
	ActionUpdate = "update"
	ActionDelete = "delete"
)

func NewActionPermission(action string) *Permission {
	return &Permission{Name: action}
}
