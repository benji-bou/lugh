package core

type Template struct {
	BaseNode[string]
	Description string `yaml:"description" json:"description"`
	Version     string `yaml:"version" json:"version"`
	Author      string `yaml:"author" json:"author"`
	path        string `yaml:"path" json:"path"`
}

func NewFileTemplate(path string) Template {
	return Template{}
}

func NewRawTemplate(raw interface{}) Template {
	return Template{}
}

// func NewMockTemplate() Template {
// 	return Template{
// 		BaseNode: BaseNode{
// 			Name:  "MockGenerator",
// 			Nodes: map[string]Nodable{
// 				"MockInput" :
// 			},
// 		},
// 	}
// }
