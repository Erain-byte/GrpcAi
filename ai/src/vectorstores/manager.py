"""
LangChain Vector Store Manager
Supports FAISS, Chroma, and Pinecone vector databases
"""

from typing import Optional, List
from langchain_core.documents import Document
from langchain_community.vectorstores import FAISS, Chroma
from langchain_core.embeddings import Embeddings

from src.embeddings.manager import get_embeddings
from config import config


class VectorStoreManager:
    """Vector Store Factory - Creates and manages vector databases"""

    @staticmethod
    def create_vector_store(
        store_type: Optional[str] = None,
        embeddings: Optional[Embeddings] = None,
        persist_directory: Optional[str] = None,
    ):
        """
        Create a vector store instance
        
        Args:
            store_type: Vector store type (faiss, chroma, pinecone)
            embeddings: Embeddings model
            persist_directory: Directory for persistence
            
        Returns:
            VectorStore instance
        """
        store_type = store_type or config.vector_store_type
        embeddings = embeddings or get_embeddings()
        persist_directory = persist_directory or config.vector_store_path

        if store_type == "faiss":
            return VectorStoreManager._create_faiss(embeddings, persist_directory)
        elif store_type == "chroma":
            return VectorStoreManager._create_chroma(embeddings, persist_directory)
        else:
            raise ValueError(f"Unsupported vector store type: {store_type}")

    @staticmethod
    def _create_faiss(embeddings: Embeddings, persist_directory: str):
        """Create FAISS vector store"""
        import os
        import faiss
        from langchain_community.docstore.in_memory import InMemoryDocstore

        # Ensure directory exists
        os.makedirs(persist_directory, exist_ok=True)

        index_path = f"{persist_directory}/faiss_index.index"

        # Try to load existing index
        if os.path.exists(index_path):
            try:
                vector_store = FAISS.load_local(
                    folder_path=persist_directory,
                    embeddings=embeddings,
                    allow_dangerous_deserialization=True,
                )
                return vector_store
            except Exception as e:
                print(f"Failed to load FAISS index: {e}, creating new one")

        # Create new FAISS index
        dimension = config.embedding_dimension
        index = faiss.IndexFlatL2(dimension)

        vector_store = FAISS(
            embedding_function=embeddings,
            index=index,
            docstore=InMemoryDocstore(),
            index_to_docstore_id={},
        )

        # Save initial empty index
        vector_store.save_local(folder_path=persist_directory)

        return vector_store

    @staticmethod
    def _create_chroma(embeddings: Embeddings, persist_directory: str):
        """Create Chroma vector store"""
        import os
        os.makedirs(persist_directory, exist_ok=True)

        vector_store = Chroma(
            embedding_function=embeddings,
            persist_directory=persist_directory,
        )

        return vector_store

    @staticmethod
    def add_documents(vector_store, documents: List[Document]):
        """
        Add documents to vector store
        
        Args:
            vector_store: VectorStore instance
            documents: List of Document objects
        """
        vector_store.add_documents(documents)

    @staticmethod
    def similarity_search(
        vector_store,
        query: str,
        k: int = 5,
    ) -> List[Document]:
        """
        Perform similarity search
        
        Args:
            vector_store: VectorStore instance
            query: Search query
            k: Number of results
            
        Returns:
            List of relevant documents
        """
        return vector_store.similarity_search(query, k=k)


# Global vector store instance
_vector_store_instance = None


def get_vector_store():
    """Get global vector store instance"""
    global _vector_store_instance
    if _vector_store_instance is None:
        _vector_store_instance = VectorStoreManager.create_vector_store()
    return _vector_store_instance
