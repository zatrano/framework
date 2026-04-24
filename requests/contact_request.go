package requests

import (
	"strings"

	"github.com/go-playground/validator/v10"
)

type ContactSubmitRequest struct {
	Name              string `form:"name"`
	Email             string `form:"email"`
	Phone             string `form:"phone"`
	Subject           string `form:"subject"`
	Message           string `form:"message"`
	TurnstileResponse string `form:"cf-turnstile-response"`
}

func (r *ContactSubmitRequest) Trim() {
	r.Name = strings.TrimSpace(r.Name)
	r.Email = strings.TrimSpace(r.Email)
	r.Phone = strings.TrimSpace(r.Phone)
	r.Subject = strings.TrimSpace(r.Subject)
	r.Message = strings.TrimSpace(r.Message)
	r.TurnstileResponse = strings.TrimSpace(r.TurnstileResponse)
}

func (r *ContactSubmitRequest) Validate() map[string]string {
	r.Trim()
	errs := make(map[string]string)
	validate := validator.New()
	if err := validate.Var(r.Name, "required,min=2,max=255"); err != nil {
		errs["Name"] = "Ad Soyad en az 2 karakter olmalıdır."
	}
	if err := validate.Var(r.Email, "required,email"); err != nil {
		errs["Email"] = "Geçerli bir e-posta adresi girin."
	}
	if r.Phone != "" {
		if err := validate.Var(r.Phone, "max=50"); err != nil {
			errs["Phone"] = "Telefon en fazla 50 karakter olabilir."
		}
	}
	if r.Subject != "" {
		if err := validate.Var(r.Subject, "max=100"); err != nil {
			errs["Subject"] = "Konu en fazla 100 karakter olabilir."
		}
	}
	if err := validate.Var(r.Message, "required,min=10,max=5000"); err != nil {
		errs["Message"] = "Mesaj en az 10, en fazla 5000 karakter olmalıdır."
	}
	return errs
}

func (r *ContactSubmitRequest) ToFormData() map[string]interface{} {
	return map[string]interface{}{
		"name":    r.Name,
		"email":   r.Email,
		"phone":   r.Phone,
		"subject": r.Subject,
		"message": r.Message,
	}
}
