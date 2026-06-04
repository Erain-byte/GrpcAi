"""
LangChain gRPC Service Implementation
Implements AiService with LangChain backend
"""

import grpc
from concurrent import futures
import logging

# Import generated protobuf code (you need to generate this first)
# from proto.ai import ai_pb2
# from proto.ai import ai_pb2_grpc

from src.chains.builder import get_conversation_chain
from src.llm.provider import get_llm
from src.memory.manager import memory_manager
from config import config


class AiServiceServicer:
    """gRPC service implementation using LangChain"""

    def Chat(self, request, context):
        """
        Standard chat RPC using LangChain
        
        Args:
            request: ChatRequest with message, conversation_id, model
            context: gRPC context
            
        Returns:
            ChatResponse with reply
        """
        try:
            # Create conversation chain
            chain = get_conversation_chain(
                conversation_id=request.conversation_id if request.conversation_id else None,
            )

            # Execute chain
            response = chain.invoke({"input": request.message})

            # Generate conversation ID if not provided
            conversation_id = request.conversation_id or f"grpc_conv_{id(response)}"

            return {
                "reply": response["response"],
                "conversation_id": conversation_id,
                "model": request.model or config.ai_model,
            }

        except Exception as e:
            logging.error(f"Chat error: {str(e)}")
            context.abort(grpc.StatusCode.INTERNAL, f"Chat failed: {str(e)}")

    def StreamChat(self, request, context):
        """
        Streaming chat RPC using LangChain
        
        Args:
            request: StreamChatRequest
            context: gRPC context
            
        Yields:
            StreamChatResponse chunks
        """
        try:
            llm = get_llm()
            
            # Create chain in streaming mode
            chain = get_conversation_chain(
                conversation_id=request.conversation_id if request.conversation_id else None,
            )

            conversation_id = request.conversation_id or f"grpc_stream_{id(llm)}"

            # Stream response
            for chunk in chain.stream({"input": request.message}):
                if "response" in chunk:
                    yield {
                        "chunk": chunk["response"],
                        "conversation_id": conversation_id,
                        "done": False,
                    }

            # Send final done signal
            yield {
                "chunk": "",
                "conversation_id": conversation_id,
                "done": True,
            }

        except Exception as e:
            logging.error(f"StreamChat error: {str(e)}")
            context.abort(grpc.StatusCode.INTERNAL, f"Stream failed: {str(e)}")

    def GetChatHistory(self, request, context):
        """
        Get conversation history RPC
        
        Args:
            request: GetChatHistoryRequest
            context: gRPC context
            
        Returns:
            GetChatHistoryResponse with messages
        """
        try:
            messages = memory_manager.get_history(
                request.conversation_id,
                limit=request.page_size * request.page if request.page_size else 50,
            )

            chat_messages = []
            for msg in messages:
                chat_messages.append({
                    "id": f"msg_{id(msg)}",
                    "role": msg.type,
                    "content": msg.content,
                    "created_at": 0,  # You can add timestamp tracking
                })

            return {
                "messages": chat_messages,
                "total": len(chat_messages),
            }

        except Exception as e:
            logging.error(f"GetChatHistory error: {str(e)}")
            context.abort(grpc.StatusCode.INTERNAL, f"History retrieval failed: {str(e)}")

    def GetChatList(self, request, context):
        """
        Get conversation list RPC
        
        Args:
            request: GetChatListRequest
            context: gRPC context
            
        Returns:
            GetChatListResponse with conversations
        """
        # TODO: Implement conversation listing from database
        # For now, return empty list
        return {
            "conversations": [],
            "total": 0,
            "page": request.page,
            "page_size": request.page_size,
        }


def serve():
    """Start gRPC server"""
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    
    # Register service
    # ai_pb2_grpc.add_AiServiceServicer_to_server(AiServiceServicer(), server)
    
    server.add_insecure_port(f"[::]:{config.grpc_port}")
    server.start()
    
    logging.info(f"gRPC server started on port {config.grpc_port}")
    server.wait_for_termination()


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    serve()
