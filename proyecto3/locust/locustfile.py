import random
import json
import logging
from locust import HttpUser, task, between, TaskSet, events
from datetime import datetime

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Países válidos
COUNTRIES = ["USA", "RUS", "CHN", "ESP", "GTM"]

# Rangos según especificación
WARPLANES_RANGE = (0, 50)
WARSHIPS_RANGE = (0, 30)


class WarReportTasks(TaskSet):
    """Tareas para enviar reportes militares"""

    def on_start(self):
        """Se ejecuta al iniciar cada usuario"""
        self.report_count = 0

    @task
    def send_report(self):
        """Envía un reporte de guerra"""
        report = {
            "country": random.choice(COUNTRIES),
            "warplanes_in_air": random.randint(*WARPLANES_RANGE),
            "warships_in_water": random.randint(*WARSHIPS_RANGE),
            "timestamp": datetime.utcnow().isoformat() + "Z"
        }

        with self.client.post(
            "/grpc-1",  # Gateway API route
            json=report,
            catch_response=True,
            timeout=10
        ) as response:
            self.report_count += 1
            
            if response.status_code == 202:  # Accepted
                response.success()
                logger.info(f"Report sent: {report['country']} (Total: {self.report_count})")
            else:
                response.failure(f"Unexpected status: {response.status_code}")
                logger.error(f"Report failed: {response.text}")


class MilitaryMonitoringUser(HttpUser):
    """Usuario simulado que envía reportes militares"""
    
    tasks = [WarReportTasks]
    wait_time = between(0.5, 2)  # Espera entre 0.5 y 2 segundos
    
    def on_start(self):
        logger.info(f"User {self.client_pool.connection_pool} started")


@events.test_start.add_listener
def on_test_start(environment, **kwargs):
    logger.info("=" * 60)
    logger.info("M.U.M.N.K8s Load Test Started")
    logger.info("Target: /grpc-1 endpoint")
    logger.info("=" * 60)


@events.test_stop.add_listener
def on_test_stop(environment, **kwargs):
    logger.info("=" * 60)
    logger.info("M.U.M.N.K8s Load Test Completed")
    logger.info(f"Total requests: {environment.stats.total_requests_per_sec}")
    logger.info("=" * 60)
