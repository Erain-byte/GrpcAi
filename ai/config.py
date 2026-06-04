"""
AI Service Configuration Loader with Pydantic Settings
Supports environment variables and .env file loading
"""

import os
from typing import Optional, List
from pydantic_settings import BaseSettings
from pydantic import Field


class AIConfig(BaseSettings):
    """AI Service Configuration with validation"""

    # ============================================
    # Service Configuration
    # ============================================
    service_name: str = Field(default="ai-service", env="SERVICE_NAME")
    service_host: str = Field(default="localhost", env="SERVICE_HOST")
    service_port: int = Field(default=8003, env="SERVICE_PORT")
    grpc_port: int = Field(default=9003, env="GRPC_PORT")
    service_version: str = Field(default="1.0.0", env="SERVICE_VERSION")

    # ============================================
    # Database Configuration (MySQL)
    # ============================================
    db_driver: str = Field(default="mysql", env="DB_DRIVER")
    db_host: str = Field(default="localhost", env="DB_HOST")
    db_port: int = Field(default=3306, env="DB_PORT")
    db_username: str = Field(default="root", env="DB_USERNAME")
    db_password: str = Field(default="", env="DB_PASSWORD")
    db_name: str = Field(default="ai_db", env="DB_NAME")
    db_max_idle_conns: int = Field(default=10, env="DB_MAX_IDLE_CONNS")
    db_max_open_conns: int = Field(default=100, env="DB_MAX_OPEN_CONNS")
    db_conn_max_lifetime: int = Field(default=3600, env="DB_CONN_MAX_LIFETIME")

    # ============================================
    # Redis Configuration
    # ============================================
    redis_host: str = Field(default="localhost", env="REDIS_HOST")
    redis_port: int = Field(default=6379, env="REDIS_PORT")
    redis_password: str = Field(default="", env="REDIS_PASSWORD")
    redis_db: int = Field(default=0, env="REDIS_DB")
    redis_pool_size: int = Field(default=100, env="REDIS_POOL_SIZE")
    redis_cluster_addresses: List[str] = Field(default_factory=list)

    # ============================================
    # JWT Configuration
    # ============================================
    jwt_secret: str = Field(default="", env="JWT_SECRET")
    jwt_expire: str = Field(default="24h", env="JWT_EXPIRE")

    # ============================================
    # Consul Configuration
    # ============================================
    consul_host: str = Field(default="localhost", env="CONSUL_HOST")
    consul_port: int = Field(default=8500, env="CONSUL_PORT")
    consul_token: str = Field(default="", env="CONSUL_TOKEN")
    consul_scheme: str = Field(default="http", env="CONSUL_SCHEME")
    consul_check_interval: str = Field(default="10s", env="CONSUL_CHECK_INTERVAL")
    consul_check_timeout: str = Field(default="5s", env="CONSUL_CHECK_TIMEOUT")
    consul_ttl: str = Field(default="30s", env="CONSUL_TTL")
    consul_deregister_critical_after: str = Field(
        default="90s", env="CONSUL_DEREGISTER_CRITICAL_AFTER"
    )
    consul_keepalive_interval: str = Field(
        default="10s", env="CONSUL_KEEPALIVE_INTERVAL"
    )
    consul_addresses: List[str] = Field(default_factory=list)

    # ============================================
    # Logger Configuration
    # ============================================
    log_level: str = Field(default="info", env="LOG_LEVEL")
    log_format: str = Field(default="json", env="LOG_FORMAT")

    # ============================================
    # CORS Configuration
    # ============================================
    cors_enabled: bool = Field(default=True, env="CORS_ENABLED")
    cors_allow_origins: List[str] = Field(default=["*"], env="CORS_ALLOW_ORIGINS")
    cors_allow_methods: List[str] = Field(
        default=["GET", "POST", "PUT", "DELETE", "OPTIONS"], env="CORS_ALLOW_METHODS"
    )
    cors_allow_headers: List[str] = Field(
        default=[
            "Origin",
            "Content-Type",
            "Accept",
            "Authorization",
            "X-Requested-With",
        ],
        env="CORS_ALLOW_HEADERS",
    )
    cors_expose_headers: List[str] = Field(
        default=["Content-Length", "Content-Type"], env="CORS_EXPOSE_HEADERS"
    )
    cors_allow_credentials: bool = Field(default=True, env="CORS_ALLOW_CREDENTIALS")
    cors_max_age: int = Field(default=12, env="CORS_MAX_AGE")

    # ============================================
    # AI Model Configuration (LangChain)
    # ============================================
    ai_model_provider: str = Field(default="openai", env="AI_MODEL_PROVIDER")
    ai_api_key: str = Field(default="", env="AI_API_KEY")
    ai_model: str = Field(default="gpt-3.5-turbo", env="AI_MODEL")
    ai_max_tokens: int = Field(default=2048, env="AI_MAX_TOKENS")
    ai_temperature: float = Field(default=0.7, env="AI_TEMPERATURE")
    
    # LangChain Advanced Config
    ai_streaming_enabled: bool = Field(default=True, env="AI_STREAMING_ENABLED")
    ai_context_window: int = Field(default=4096, env="AI_CONTEXT_WINDOW")
    ai_top_p: float = Field(default=1.0, env="AI_TOP_P")
    ai_frequency_penalty: float = Field(default=0.0, env="AI_FREQUENCY_PENALTY")
    ai_presence_penalty: float = Field(default=0.0, env="AI_PRESENCE_PENALTY")
    
    # Embedding Model Config
    embedding_model: str = Field(default="text-embedding-ada-002", env="EMBEDDING_MODEL")
    embedding_dimension: int = Field(default=1536, env="EMBEDDING_DIMENSION")
    
    # Vector Store Config
    vector_store_type: str = Field(default="faiss", env="VECTOR_STORE_TYPE")
    vector_store_path: str = Field(default="./data/vectors", env="VECTOR_STORE_PATH")

    # ============================================
    # Shutdown Configuration
    # ============================================
    shutdown_timeout: str = Field(default="5s", env="SHUTDOWN_TIMEOUT")

    # ============================================
    # TLS/mTLS Configuration (Optional)
    # ============================================
    grpc_use_tls: bool = Field(default=False, env="GRPC_USE_TLS")
    grpc_insecure_skip_verify: bool = Field(
        default=False, env="GRPC_INSECURE_SKIP_VERIFY"
    )
    grpc_cert_file: str = Field(default="", env="GRPC_CERT_FILE")
    grpc_key_file: str = Field(default="", env="GRPC_KEY_FILE")
    grpc_ca_file: str = Field(default="", env="GRPC_CA_FILE")
    grpc_server_name: str = Field(default="ai-service", env="GRPC_SERVER_NAME")

    class Config:
        env_file = ".env"
        env_file_encoding = "utf-8"
        case_sensitive = False

    def validate(self):
        """Validate required configuration"""
        errors = []

        if not self.jwt_secret:
            errors.append("JWT_SECRET is required")

        if self.ai_model_provider == "openai" and not self.ai_api_key:
            errors.append("AI_API_KEY is required when using OpenAI provider")

        if errors:
            raise ValueError(f"Configuration validation failed:\n" + "\n".join(errors))

    def get_database_url(self) -> str:
        """Get database connection URL"""
        return (
            f"{self.db_driver}://{self.db_username}:{self.db_password}"
            f"@{self.db_host}:{self.db_port}/{self.db_name}"
        )

    def get_redis_url(self) -> str:
        """Get Redis connection URL"""
        if self.redis_cluster_addresses:
            return ",".join(self.redis_cluster_addresses)

        password_part = f":{self.redis_password}@" if self.redis_password else ""
        return f"redis://{password_part}{self.redis_host}:{self.redis_port}/{self.redis_db}"


# Global configuration instance
config = AIConfig()


if __name__ == "__main__":
    # Test configuration loading
    try:
        config.validate()
        print("✅ Configuration loaded successfully!")
        print(f"Service: {config.service_name}")
        print(f"Database: {config.get_database_url()}")
        print(f"Redis: {config.get_redis_url()}")
        print(f"AI Model: {config.ai_model}")
        print(f"Provider: {config.ai_model_provider}")
        print(f"Vector Store: {config.vector_store_type}")
        print(f"TLS Enabled: {config.grpc_use_tls}")
    except ValueError as e:
        print(f"❌ Configuration error: {e}")
