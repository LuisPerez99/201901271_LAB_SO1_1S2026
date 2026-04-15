use reqwest::Client;
use std::env;

/// Configuración y estado global de la aplicación
pub struct AppState {
    pub go_api_client_url: String,
    pub http_client: Client,
}

impl AppState {
    pub fn new() -> Self {
        let go_api_client_url = env::var("GO_API_CLIENT_URL")
            .unwrap_or_else(|_| "http://go-api-client:8080/report".to_string());

        AppState {
            go_api_client_url,
            http_client: Client::new(),
        }
    }
}
