"""
LangChain Chain Builder
Provides pre-built conversation chains with memory support
"""

from typing import Optional
from langchain.chains import ConversationChain
from langchain_core.prompts import ChatPromptTemplate, MessagesPlaceholder
from langchain_core.language_models import BaseChatModel

from src.llm.provider import get_llm
from src.memory.manager import memory_manager


class ChainBuilder:
    """Build and manage LangChain conversation chains"""

    @staticmethod
    def create_conversation_chain(
        llm: Optional[BaseChatModel] = None,
        conversation_id: Optional[str] = None,
        system_prompt: Optional[str] = None,
        use_summary: bool = False,
        window_size: Optional[int] = None,
    ) -> ConversationChain:
        """
        Create a conversation chain with memory
        
        Args:
            llm: LLM instance (uses global if not provided)
            conversation_id: Conversation ID for memory persistence
            system_prompt: Custom system prompt
            use_summary: Use summary memory instead of buffer
            window_size: Window size for window memory (None = no limit)
            
        Returns:
            ConversationChain instance
        """
        llm = llm or get_llm()

        # Create custom prompt template
        if system_prompt:
            prompt_template = ChatPromptTemplate.from_messages([
                ("system", system_prompt),
                MessagesPlaceholder(variable_name="history"),
                ("human", "{input}"),
            ])
        else:
            # Default prompt
            prompt_template = ChatPromptTemplate.from_messages([
                ("system", "You are a helpful AI assistant."),
                MessagesPlaceholder(variable_name="history"),
                ("human", "{input}"),
            ])

        # Create memory based on configuration
        if conversation_id:
            if use_summary:
                memory = memory_manager.create_summary_memory(
                    llm=llm,
                    conversation_id=conversation_id,
                )
            elif window_size:
                memory = memory_manager.create_window_memory(
                    conversation_id=conversation_id,
                    k=window_size,
                )
            else:
                memory = memory_manager.create_buffer_memory(
                    conversation_id=conversation_id,
                )
        else:
            # In-memory only (no persistence)
            from langchain.memory import ConversationBufferMemory
            memory = ConversationBufferMemory(
                memory_key="chat_history",
                return_messages=True,
            )

        # Create chain
        chain = ConversationChain(
            llm=llm,
            memory=memory,
            prompt=prompt_template,
            verbose=True,  # Enable for debugging
        )

        return chain

    @staticmethod
    def create_simple_chain(
        llm: Optional[BaseChatModel] = None,
        system_prompt: str = "You are a helpful assistant.",
    ):
        """
        Create a simple chain without memory (stateless)
        
        Args:
            llm: LLM instance
            system_prompt: System prompt
            
        Returns:
            Runnable sequence
        """
        from langchain_core.output_parsers import StrOutputParser
        
        llm = llm or get_llm()
        
        prompt = ChatPromptTemplate.from_messages([
            ("system", system_prompt),
            ("human", "{input}"),
        ])
        
        chain = prompt | llm | StrOutputParser()
        return chain


# Convenience function
def get_conversation_chain(conversation_id: str = None, **kwargs):
    """Quick access to conversation chain"""
    return ChainBuilder.create_conversation_chain(
        conversation_id=conversation_id,
        **kwargs
    )
