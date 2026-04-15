use actix_web::{web, HttpResponse, Result as ActixResult};
use chrono::Utc;
use log::{error, info};
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

            let forward_result = app_state
                .http_client
                .post(&app_state.go_api_client_url)
                .json(&*report)
                .send()
                .await;

            match forward_result {
                Ok(resp) if resp.status().is_success() => {
                    info!(
                        "Forwarded report {} to go-api-client successfully",
                        validated.report_id
                    );
                    Ok(HttpResponse::Accepted().json(response))
                }
                Ok(resp) => {
                    error!(
                        "go-api-client rejected report {} with status {}",
                        validated.report_id,
                        resp.status()
                    );
                    Ok(HttpResponse::BadGateway().json(serde_json::json!({
                        "status": "error",
                        "message": "Failed to forward report to go-api-client",
                        "upstream_status": resp.status().as_u16(),
                        "report_id": validated.report_id,
                        "timestamp": Utc::now().to_rfc3339(),
                    })))
                }
                Err(err) => {
                    error!(
                        "Forward error for report {}: {}",
                        validated.report_id,
                        err
                    );
                    Ok(HttpResponse::BadGateway().json(serde_json::json!({
                        "status": "error",
                        "message": "go-api-client unavailable",
                        "details": err.to_string(),
                        "report_id": validated.report_id,
                        "timestamp": Utc::now().to_rfc3339(),
                    })))
                }
            }
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
