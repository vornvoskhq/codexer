package types

type ExecStatusResponse struct {
	Reasoning       string `json:"reasoning"`
	SubtaskFinished bool   `json:"subtaskFinished"`
}
