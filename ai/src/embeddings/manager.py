"""
LangChain Embedding Manager
Supports multiple embedding models for vector search
"""

from typing import Optional
from langchain_core.embeddings import Embeddings
from langchain_openai import OpenAIEmbeddings
from langchain_community.embeddings import HuggingFaceEmbeddings

from config import config


class EmbeddingManager:
    """Embedding Model Factory - Creates embedding models"""

    @staticmethod
    def create_embeddings(
        provider: Optional[str] = None,
        model: Optional[str] = None,
    ) -> Embeddings:
        """
        Create an embedding model instance
        
        Args:
            provider: Embedding provider (openai, huggingface)
            model: Model name
            
        Returns:
            Embeddings instance
        """
        provider = provider or "openai"  # Default to OpenAI
        model = model or config.embedding_model

        if provider == "openai":
            return EmbeddingManager._create_openai_embeddings(model)
        elif provider == "huggingface":
            return EmbeddingManager._create_huggingface_embeddings(model)
        else:
            raise ValueError(f"Unsupported embedding provider: {provider}")

    @staticmethod
    def _create_openai_embeddings(model: str) -> OpenAIEmbeddings:
        """Create OpenAI embeddings"""
        return OpenAIEmbeddings(
            model=model,
            openai_api_key=config.ai_api_key,
            dimensions=config.embedding_dimension,
        )

    @staticmethod
    def _create_huggingface_embeddings(model: str) -> HuggingFaceEmbeddings:
        """Create HuggingFace embeddings (local)"""
        return HuggingFaceEmbeddings(
            model_name=model,
            model_kwargs={"device": "cpu"},  # Use "cuda" for GPU
            encode_kwargs={"normalize_embeddings": True},
        )

    @staticmethod
    def get_available_providers() -> list[str]:
        """Get list of available embedding providers"""
        return ["openai", "huggingface"]


# Global embeddings instance (lazy loading)
_embeddings_instance: Optional[Embeddings] = None


def get_embeddings() -> Embeddings:
    """
    Get global embeddings instance (singleton pattern)
    
    Returns:
        Embeddings instance
    """
    global _embeddings_instance
    if _embeddings_instance is None:
        _embeddings_instance = EmbeddingManager.create_embeddings()
    return _embeddings_instance


def reset_embeddings():
    """Reset global embeddings instance"""
    global _embeddings_instance
    _embeddings_instance = None
