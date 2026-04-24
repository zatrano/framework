package oauth

import (
	"os"

	"github.com/zatrano/framework/configs/logconfig"
	"github.com/zatrano/framework/services"
)

type ProviderFactory struct{}

func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{}
}

func (f *ProviderFactory) CreateOAuthHandler(authService services.IAuthService) *OAuthHandler {
	handler := NewOAuthHandler(authService)

	if googleProvider := f.CreateGoogleProvider(); googleProvider != nil {
		handler.RegisterProvider(googleProvider)
	}

	// İleride diğer provider'lar buraya eklenecek
	// if facebookProvider := f.CreateFacebookProvider(); facebookProvider != nil {
	//     handler.RegisterProvider(facebookProvider)
	// }

	// if githubProvider := f.CreateGitHubProvider(); githubProvider != nil {
	//     handler.RegisterProvider(githubProvider)
	// }

	return handler
}

func (f *ProviderFactory) CreateGoogleProvider() *GoogleProvider {
	if os.Getenv("GOOGLE_CLIENT_ID") == "" ||
		os.Getenv("GOOGLE_CLIENT_SECRET") == "" ||
		os.Getenv("GOOGLE_REDIRECT_URI") == "" {
		logconfig.Log.Warn("Google OAuth environment variables eksik, Google OAuth devre dışı")
		return nil
	}

	return NewGoogleProvider()
}
