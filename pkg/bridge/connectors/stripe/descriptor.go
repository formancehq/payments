package stripe

type TaskDescriptor struct {
	Main    bool   `json:"main,omitempty"`
	Account string `json:"account,omitempty"`
}
