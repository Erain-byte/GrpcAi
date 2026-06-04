"""
AI Service Main Application
FastAPI + LangChain + gRPC integration
"""

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
import uvicorn
import logging

from config import config
from src.routes.langchain_routes import router as langchain_router


def create_app() -> FastAPI:
    """Create and configure FastAPI application"""
    
    app = FastAPI(
        title=config.service_name,
        version=config.service_version,
        description="AI Service with LangChain integration",
    )

    # CORS middleware
    if config.cors_enabled:
        app.add_middleware(
            CORSMiddleware,
            allow_origins=config.cors_allow_origins,
            allow_methods=config.cors_allow_methods,
            allow_headers=config.cors_allow_headers,
            allow_credentials=config.cors_allow_credentials,
            expose_headers=config.cors_expose_headers,
            max_age=config.cors_max_age,
        )

    # Register routers
    app.include_router(langchain_router)

    # Health check endpoint
    @app.get("/health")
    def health_check():
        return {
            "status": "healthy",
            "service": config.service_name,
            "version": config.service_version,
        }

    return app


# Create app instance
app = create_app()


if __name__ == "__main__":
    # Configure logging
    logging.basicConfig(
        level=getattr(logging, config.log_level.upper()),
        format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
    )

    # Validate configuration
    try:
        config.validate()
        logging.info("✅ Configuration validated successfully")
    except ValueError as e:
        logging.error(f"❌ Configuration validation failed: {e}")
        exit(1)

    # Start server
    logging.info(f"🚀 Starting {config.service_name} on {config.service_host}:{config.service_port}")
    uvicorn.run(
        app,
        host=config.service_host,
        port=config.service_port,
        log_level=config.log_level,
    )
