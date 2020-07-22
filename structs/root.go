package structs

//Configs is the struct used to store all the common configs that are used everywhere
//the struct field tags are added to ensure that when json marshal happens these keys are used instead of the struct key
//https://golang.org/pkg/encoding/json/#Marshal
type Configs struct {
	AuthServiceKey string   `json:"auth"`
	Hostnames      []string `json:"hostnames"`
	Validity       int      `json:"validity"`
}
