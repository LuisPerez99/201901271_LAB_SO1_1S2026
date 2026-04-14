use actix_web::{web, HttpResponse, Result as ActixResult};
use chrono::Utc;
use log::{info, error};
use std::sync::Arc;

use crate::models::{WarReport, validate_report, HealthResponse, ReportResponse};
use crate::config::AppState;

/// Endpoint GET /health - Health check
pub async fn health_check() -> ActixResult<HttpResponse> {
    info!("Health check requested");
    
    let response = HealthResponse {
        status: "healthy".to_string(),
        timestamp: Utc::now().to_rfc3339(),
        version: env!("CARGO_PKG_VERSION").to_string(),
    };

    Ok(HttpResponse::Ok().json(response))
}

/// Endpoint POST /report - Recibe reportes militares
pub async fn receive_report(
    app_state: web::Data<Arc<AppState>>,
    report: web::Json<WarReport>,
) -> ActixResult<HttpResponse> {
    info!("Received report: {:?}", report);

    // Validar reporte
    match validate_report(&report) {
        Ok(validated) => {
            info!(
                "Report validated and forwarded: country={}, id={}",
                validated.country, validated.report_id
            );

            let response = ReportResponse {
                status: "accepted".to_string(),
                message: "Report submitted for processing".to_string(),
                report_id: validated.report_id.clone(),
                timestamp: Utc::now().to_rfc3339(),
            };

            // TODO: Forward to Go Service 1 via gRPC or HTTP
            // For now, just log and return

            Ok(HttpResponse::Accepted().json(response))
        }
        Err(e) => {
            error!("Validation error: {}", e);
            Ok(HttpResponse::BadRequest().json(serde_json::json!({
                "status": "error",
                "message": e,
                "timestamp": Utc::now().to_rfc3339(),
            })))
        }
    }
}
