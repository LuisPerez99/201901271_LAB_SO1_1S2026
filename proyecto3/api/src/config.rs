use std::sync::Arc;

/// Configuración y estado global de la aplicación
pub struct AppState {
    pub name: String,
    pub version: String,
}

impl AppState {
    pub fn new() -> Self {
        AppState {
            name: "M.U.M.N.K8s REST API".to_string(),
            version: env!("CARGO_PKG_VERSION").to_string(),
        }
    }
}
