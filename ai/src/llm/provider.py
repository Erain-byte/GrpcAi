"""
LangChain LLM Provider Manager
Supports multiple LLM providers: OpenAI, Anthropic, Google Gemini, etc.
"""

from typing import Optional
from langchain_core.language_models import BaseChatModel
from langchain_openai import ChatOpenAI
from langchain_anthropic import ChatAnthropic
from langchain_google_genai import ChatGoogleGenerativeAI

from config import config


class LLMProvider:
    """LLM Provider Factory - Creates chat models based on configuration"""

    @staticmethod
    def create_llm(
        provider: Optional[str] = None,
        model: Optional[str] = None,
        temperature: Optional[float] = None,
        max_tokens: Optional[int] = None,
        streaming: bool = True,
    ) -> BaseChatModel:
        """
        Create a chat model instance based on provider configuration
        
        Args:
            provider: LLM provider (openai, anthropic, google)
            model: Model name
            temperature: Sampling temperature (0.0-1.0)
            max_tokens: Maximum tokens in response
            streaming: Enable streaming mode
            
        Returns:
            BaseChatModel instance
        """
        provider = provider or config.ai_model_provider
        model = model or config.ai_model
        temperature = temperature if temperature is not None else config.ai_temperature
        max_tokens = max_tokens or config.ai_max_tokens

        if provider == "openai":
            return LLMProvider._create_openai(model, temperature, max_tokens, streaming)
        elif provider == "anthropic":
            return LLMProvider._create_anthropic(model, temperature, max_tokens, streaming)
        elif provider == "google":
            return LLMProvider._create_google(model, temperature, max_tokens, streaming)
        else:
            raise ValueError(f"Unsupported LLM provider: {provider}")

    @staticmethod
    def _create_openai(
        model: str, temperature: float, max_tokens: int, streaming: bool
    ) -> ChatOpenAI:
        """Create OpenAI chat model"""
        return ChatOpenAI(
            model=model,
            temperature=temperature,
            max_tokens=max_tokens,
            streaming=streaming,
            openai_api_key=config.ai_api_key,
            top_p=config.ai_top_p,
            frequency_penalty=config.ai_frequency_penalty,
            presence_penalty=config.ai_presence_penalty,
        )

    @staticmethod
    def _create_anthropic(
        model: str, temperature: float, max_tokens: int, streaming: bool
    ) -> ChatAnthropic:
        """Create Anthropic Claude chat model"""
        return ChatAnthropic(
            model=model,
            temperature=temperature,
            max_tokens_to_sample=max_tokens,
            streaming=streaming,
            anthropic_api_key=config.ai_api_key,
        )

    @staticmethod
    def _create_google(
        model: str, temperature: float, max_tokens: int, streaming: bool
    ) -> ChatGoogleGenerativeAI:
        """Create Google Gemini chat model"""
        return ChatGoogleGenerativeAI(
            model=model,
            temperature=temperature,
            max_output_tokens=max_tokens,
            streaming=streaming,
            google_api_key=config.ai_api_key,
            top_p=config.ai_top_p,
        )

    @staticmethod
    def get_available_providers() -> list[str]:
        """Get list of available LLM providers"""
        return ["openai", "anthropic", "google"]


# Global LLM instance (lazy loading)
_llm_instance: Optional[BaseChatModel] = None


def get_llm() -> BaseChatModel:
    """
    Get global LLM instance (singleton pattern)
    
    Returns:
        BaseChatModel instance
    """
    global _llm_instance
    if _llm_instance is None:
        _llm_instance = LLMProvider.create_llm()
    return _llm_instance


def reset_llm():
    """Reset global LLM instance (useful for testing)"""
    global _llm_instance
    _llm_instance = None
