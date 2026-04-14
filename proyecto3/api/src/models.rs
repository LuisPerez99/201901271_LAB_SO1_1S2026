use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};

/// Estructura para recibir reportes JSON de Locust
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WarReport {
    pub country: String,
    pub warplanes_in_air: i32,
    pub warships_in_water: i32,
    pub timestamp: String,
}

/// Estructura para validar y enriquecer el reporte
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidatedReport {
    pub country: String,
    pub warplanes_in_air: i32,
    pub warships_in_water: i32,
    pub timestamp: String,
    pub received_at: String,
    pub report_id: String,
}

/// Response para endpoint /report
#[derive(Debug, Serialize, Deserialize)]
pub struct ReportResponse {
    pub status: String,
    pub message: String,
    pub report_id: String,
    pub timestamp: String,
}

/// Response para endpoint /health
#[derive(Debug, Serialize, Deserialize)]
pub struct HealthResponse {
    pub status: String,
    pub timestamp: String,
    pub version: String,
}

/// Enum para países válidos
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Country {
    Unknown = 0,
    USA = 1,
    RUS = 2,
    CHN = 3,
    ESP = 4,
    GTM = 5,
}

impl Country {
    pub fn from_string(s: &str) -> Option<Self> {
        match s.to_uppercase().as_str() {
            "USA" => Some(Country::USA),
            "RUS" => Some(Country::RUS),
            "CHN" => Some(Country::CHN),
            "ESP" => Some(Country::ESP),
            "GTM" => Some(Country::GTM),
            _ => None,
        }
    }

    pub fn to_i32(self) -> i32 {
        self as i32
    }
}

/// Validar estructura del reporte
pub fn validate_report(report: &WarReport) -> Result<ValidatedReport, String> {
    // Validar país
    if Country::from_string(&report.country).is_none() {
        return Err(format!("Invalid country: {}", report.country));
    }

    // Validar rangos
    if report.warplanes_in_air < 0 || report.warplanes_in_air > 50 {
        return Err(format!(
            "Invalid warplanes_in_air: {} (should be 0-50)",
            report.warplanes_in_air
        ));
    }

    if report.warships_in_water < 0 || report.warships_in_water > 30 {
        return Err(format!(
            "Invalid warships_in_water: {} (should be 0-30)",
            report.warships_in_water
        ));
    }

    // Generar ID único
    let report_id = uuid::Uuid::new_v4().to_string();
    
    Ok(ValidatedReport {
        country: report.country.clone(),
        warplanes_in_air: report.warplanes_in_air,
        warships_in_water: report.warships_in_water,
        timestamp: report.timestamp.clone(),
        received_at: Utc::now().to_rfc3339(),
        report_id,
    })
}
