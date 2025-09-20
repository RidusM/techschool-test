package entity

type Delivery struct {
	Name    string `json:"name"    validate:"required,max=100"`
	Phone   string `json:"phone"   validate:"required,e164"`
	Zip     string `json:"zip"     validate:"required,max=20"`
	City    string `json:"city"    validate:"required,max=100"`
	Address string `json:"address" validate:"required,max=500"`
	Region  string `json:"region"  validate:"required,max=100"`
	Email   string `json:"email"   validate:"required,email,max=100"`
}
