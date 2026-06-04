"""
LangChain Agent Builder
Create agents with tools for complex task execution
"""

from typing import Optional, List
from langchain.agents import create_react_agent, AgentExecutor
from langchain_core.tools import Tool
from langchain_core.language_models import BaseChatModel

from src.llm.provider import get_llm


class AgentBuilder:
    """Build LangChain agents with custom tools"""

    @staticmethod
    def create_basic_agent(
        llm: Optional[BaseChatModel] = None,
        tools: Optional[List[Tool]] = None,
        system_prompt: Optional[str] = None,
    ) -> AgentExecutor:
        """
        Create a ReAct agent with tools
        
        Args:
            llm: LLM instance
            tools: List of available tools
            system_prompt: Custom system prompt
            
        Returns:
            AgentExecutor instance
        """
        llm = llm or get_llm()
        tools = tools or []

        # Default system prompt for ReAct agent
        if not system_prompt:
            system_prompt = """You are a helpful AI assistant with access to various tools.

When you need to use a tool:
1. Think about which tool is appropriate
2. Use the tool with proper arguments
3. Analyze the tool's output
4. Provide a clear response to the user

Always be helpful and concise."""

        # Create ReAct agent
        agent = create_react_agent(
            llm=llm,
            tools=tools,
            state_modifier=system_prompt,
        )

        # Wrap in executor
        agent_executor = AgentExecutor(
            agent=agent,
            tools=tools,
            verbose=True,
            handle_parsing_errors=True,
            max_iterations=5,
        )

        return agent_executor

    @staticmethod
    def create_search_agent(
        llm: Optional[BaseChatModel] = None,
        enable_web_search: bool = True,
        enable_wikipedia: bool = True,
    ) -> AgentExecutor:
        """
        Create an agent with search capabilities
        
        Args:
            llm: LLM instance
            enable_web_search: Enable web search tool
            enable_wikipedia: Enable Wikipedia lookup
            
        Returns:
            AgentExecutor with search tools
        """
        from langchain_community.tools import DuckDuckGoSearchRun
        from langchain_community.tools.wikipedia.tool import WikipediaQueryRun
        from langchain_community.utilities import WikipediaAPIWrapper

        tools = []

        if enable_web_search:
            search_tool = DuckDuckGoSearchRun()
            tools.append(
                Tool(
                    name="web_search",
                    func=search_tool.run,
                    description="Search the web for current information. Use this when you need up-to-date information.",
                )
            )

        if enable_wikipedia:
            wikipedia = WikipediaQueryRun(api_wrapper=WikipediaAPIWrapper())
            tools.append(
                Tool(
                    name="wikipedia",
                    func=wikipedia.run,
                    description="Search Wikipedia for factual information. Use this for historical facts, definitions, and well-known topics.",
                )
            )

        return AgentBuilder.create_basic_agent(llm=llm, tools=tools)


# Convenience function
def get_search_agent(**kwargs):
    """Quick access to search-enabled agent"""
    return AgentBuilder.create_search_agent(**kwargs)
