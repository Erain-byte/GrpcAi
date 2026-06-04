"""
LangChain Chat Service - FastAPI Routes
Integrates LangChain with gRPC and HTTP endpoints
"""

from fastapi import APIRouter, HTTPException
from pydantic import BaseModel
from typing import Optional, AsyncGenerator
import asyncio

from src.chains.builder import get_conversation_chain
from src.llm.provider import get_llm
from src.memory.manager import memory_manager

router = APIRouter(prefix="/langchain", tags=["langchain"])


class ChatRequest(BaseModel):
    """Chat request model"""
    message: str
    conversation_id: Optional[str] = None
    model: Optional[str] = None
    use_summary_memory: bool = False
    window_size: Optional[int] = None


class ChatResponse(BaseModel):
    """Chat response model"""
    reply: str
    conversation_id: str
    model: str


class StreamChatRequest(BaseModel):
    """Stream chat request model"""
    message: str
    conversation_id: Optional[str] = None
    model: Optional[str] = None


class StreamChunk(BaseModel):
    """Stream chunk model"""
    chunk: str
    conversation_id: str
    done: bool = False


@router.post("/chat", response_model=ChatResponse)
async def chat(request: ChatRequest):
    """
    Standard chat endpoint using LangChain
    
    Args:
        request: Chat request with message and optional conversation_id
        
    Returns:
        ChatResponse with AI reply
    """
    try:
        # Create conversation chain with memory
        chain = get_conversation_chain(
            conversation_id=request.conversation_id,
            use_summary=request.use_summary_memory,
            window_size=request.window_size,
        )

        # Execute chain
        response = chain.invoke({"input": request.message})

        # Generate or use existing conversation ID
        conversation_id = request.conversation_id or f"conv_{id(response)}"

        return ChatResponse(
            reply=response["response"],
            conversation_id=conversation_id,
            model=request.model or "default",
        )

    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Chat error: {str(e)}")


@router.post("/stream-chat")
async def stream_chat(request: StreamChatRequest) -> AsyncGenerator[str, None]:
    """
    Streaming chat endpoint using LangChain
    
    Args:
        request: Stream chat request
        
    Yields:
        SSE-formatted chunks
    """
    try:
        llm = get_llm()
        
        # Create chain (streaming mode)
        chain = get_conversation_chain(
            conversation_id=request.conversation_id,
        )

        # Stream response
        conversation_id = request.conversation_id or f"conv_stream_{id(llm)}"
        
        async for chunk in chain.astream({"input": request.message}):
            if "response" in chunk:
                yield f"data: {StreamChunk(chunk=chunk['response'], conversation_id=conversation_id).model_dump_json()}\n\n"
                await asyncio.sleep(0.01)  # Small delay for streaming effect

        # Send done signal
        yield f"data: {StreamChunk(chunk='', conversation_id=conversation_id, done=True).model_dump_json()}\n\n"

    except Exception as e:
        yield f"data: {{'error': '{str(e)}'}}\n\n"


@router.get("/history/{conversation_id}")
async def get_chat_history(conversation_id: str, limit: int = 50):
    """
    Get conversation history from Redis
    
    Args:
        conversation_id: Conversation ID
        limit: Maximum messages to retrieve
        
    Returns:
        List of messages
    """
    try:
        messages = memory_manager.get_history(conversation_id, limit=limit)
        return {
            "conversation_id": conversation_id,
            "messages": [
                {"role": msg.type, "content": msg.content}
                for msg in messages
            ],
            "total": len(messages),
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"History retrieval error: {str(e)}")


@router.delete("/conversation/{conversation_id}")
async def delete_conversation(conversation_id: str):
    """
    Delete conversation history
    
    Args:
        conversation_id: Conversation ID to delete
        
    Returns:
        Success message
    """
    try:
        memory_manager.clear_conversation(conversation_id)
        return {"message": f"Conversation {conversation_id} deleted"}
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Deletion error: {str(e)}")


@router.get("/models")
async def list_models():
    """
    List available LLM models
    
    Returns:
        Available providers and models
    """
    from src.llm.provider import LLMProvider
    
    return {
        "providers": LLMProvider.get_available_providers(),
        "current_provider": config.ai_model_provider,
        "current_model": config.ai_model,
    }
