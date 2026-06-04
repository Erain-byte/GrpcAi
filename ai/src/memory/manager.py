"""
LangChain Memory Manager
Supports conversation history management with Redis persistence
"""

from typing import Optional, List
from langchain_core.messages import BaseMessage, HumanMessage, AIMessage
from langchain.memory import (
    ConversationBufferMemory,
    ConversationSummaryMemory,
    ConversationBufferWindowMemory,
)
import redis
import json
from config import config


class MemoryManager:
    """Conversation Memory Manager with Redis persistence"""

    def __init__(self):
        self.redis_client = redis.Redis(
            host=config.redis_host,
            port=config.redis_port,
            password=config.redis_password if config.redis_password else None,
            db=config.redis_db,
            decode_responses=True,
        )

    def create_buffer_memory(
        self,
        conversation_id: str,
        max_token_limit: int = 2000,
        return_messages: bool = True,
    ) -> ConversationBufferMemory:
        """
        Create conversation buffer memory
        
        Args:
            conversation_id: Unique conversation identifier
            max_token_limit: Maximum tokens to keep in memory
            return_messages: Return message objects instead of strings
            
        Returns:
            ConversationBufferMemory instance
        """
        memory = ConversationBufferMemory(
            memory_key="chat_history",
            return_messages=return_messages,
            max_token_limit=max_token_limit,
        )
        
        # Load existing history from Redis
        history = self._load_history(conversation_id)
        for msg in history:
            memory.chat_memory.add_message(msg)
        
        return memory

    def create_summary_memory(
        self,
        llm,
        conversation_id: str,
        max_token_limit: int = 2000,
    ) -> ConversationSummaryMemory:
        """
        Create conversation summary memory (summarizes long conversations)
        
        Args:
            llm: LLM instance for summarization
            conversation_id: Unique conversation identifier
            max_token_limit: Maximum tokens before summarization
            
        Returns:
            ConversationSummaryMemory instance
        """
        memory = ConversationSummaryMemory.from_llm(
            llm=llm,
            memory_key="chat_history",
            max_token_limit=max_token_limit,
        )
        
        # Load existing history
        history = self._load_history(conversation_id)
        for msg in history:
            memory.chat_memory.add_message(msg)
        
        return memory

    def create_window_memory(
        self,
        conversation_id: str,
        k: int = 10,
        return_messages: bool = True,
    ) -> ConversationBufferWindowMemory:
        """
        Create window memory (keeps last K messages)
        
        Args:
            conversation_id: Unique conversation identifier
            k: Number of recent messages to keep
            return_messages: Return message objects
            
        Returns:
            ConversationBufferWindowMemory instance
        """
        memory = ConversationBufferWindowMemory(
            memory_key="chat_history",
            k=k,
            return_messages=return_messages,
        )
        
        # Load existing history
        history = self._load_history(conversation_id)
        for msg in history[-k:]:  # Only keep last K messages
            memory.chat_memory.add_message(msg)
        
        return memory

    def save_message(self, conversation_id: str, role: str, content: str):
        """
        Save a single message to Redis
        
        Args:
            conversation_id: Conversation ID
            role: Message role (human/ai)
            content: Message content
        """
        key = f"conversation:{conversation_id}:messages"
        message = {
            "role": role,
            "content": content,
        }
        self.redis_client.rpush(key, json.dumps(message))
        
        # Set TTL (30 days)
        self.redis_client.expire(key, 30 * 24 * 3600)

    def get_history(self, conversation_id: str, limit: int = 50) -> List[BaseMessage]:
        """
        Get conversation history from Redis
        
        Args:
            conversation_id: Conversation ID
            limit: Maximum number of messages to retrieve
            
        Returns:
            List of BaseMessage objects
        """
        key = f"conversation:{conversation_id}:messages"
        messages_json = self.redis_client.lrange(key, -limit, -1)
        
        messages = []
        for msg_json in messages_json:
            msg_data = json.loads(msg_json)
            if msg_data["role"] == "human":
                messages.append(HumanMessage(content=msg_data["content"]))
            elif msg_data["role"] == "ai":
                messages.append(AIMessage(content=msg_data["content"]))
        
        return messages

    def _load_history(self, conversation_id: str) -> List[BaseMessage]:
        """Load full conversation history"""
        return self.get_history(conversation_id, limit=1000)

    def clear_conversation(self, conversation_id: str):
        """Clear all messages for a conversation"""
        key = f"conversation:{conversation_id}:messages"
        self.redis_client.delete(key)

    def delete_conversation(self, conversation_id: str):
        """Delete entire conversation (alias for clear)"""
        self.clear_conversation(conversation_id)


# Global memory manager instance
memory_manager = MemoryManager()
