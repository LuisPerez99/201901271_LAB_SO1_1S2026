use actix_web::{middleware, web, App, HttpServer};
use log::info;
use std::sync::Arc;

mod models;
mod handlers;
mod config;

use handlers::{health_check, receive_report};
use config::AppState;

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    env_logger::init_from_env(env_logger::Env::new().default_filter_or("info"));
    
    info!("Starting Rust REST API server...");
    
    let app_state = Arc::new(AppState::new());
    
    info!("Server listening on 0.0.0.0:3000");
    
    HttpServer::new(move || {
        App::new()
            .app_data(web::Data::new(app_state.clone()))
            .wrap(middleware::Logger::default())
            .route("/health", web::get().to(health_check))
            .route("/report", web::post().to(receive_report))
            .route("/grpc-201901271", web::post().to(receive_report))
    })
    .bind("0.0.0.0:3000")?
    .run()
    .await
}
