package commands

// Internal command name constants (for LLM/system invocation)
const (
	// Basic commands
	CmdNameExit    = "command_exit"
	CmdNameQuit    = "command_quit"
	CmdNameQ       = "command_q"
	CmdNameClear   = "command_clear"
	CmdNameCompact = "command_compact"
	CmdNameHelp    = "command_help"
	CmdNameMemory  = "command_memory"
	CmdNameInit    = "command_init"

	// Session commands
	CmdNameSessions      = "command_sessions"
	CmdNameSessionCreate = "command_session_create"
	CmdNameSessionShow   = "command_session_show"
	CmdNameSessionDelete = "command_session_delete"
	CmdNameSessionUpdate = "command_session_update"

	// Profile commands
	CmdNameProfile       = "command_profile"
	CmdNameProfileShow   = "command_profile_show"
	CmdNameProfileImport = "command_profile_import"

	// Config commands
	CmdNameConfig     = "command_config"
	CmdNameConfigShow = "command_config_show"
	CmdNameConfigPath = "command_config_path"
	CmdNameConfigGet  = "command_config_get"

	// Knowledge commands
	CmdNameKnowledge       = "command_knowledge"
	CmdNameKnowledgeSearch = "command_knowledge_search"
	CmdNameKnowledgeImport = "command_knowledge_import"
	CmdNameKnowledgeStats  = "command_knowledge_stats"

	// Subscription commands
	CmdNameSubscription       = "command_subscription"
	CmdNameSubscriptionShow   = "command_subscription_show"
	CmdNameSubscriptionAdd    = "command_subscription_add"
	CmdNameSubscriptionDelete = "command_subscription_delete"

	// Skill management commands
	CmdNameSkill       = "command_skill"
	CmdNameSkillCreate = "command_skill_create"
	CmdNameSkillUpdate = "command_skill_update"
	CmdNameSkillDelete = "command_skill_delete"
	CmdNameSkillReload = "command_skill_reload"
	CmdNameSkillList   = "command_skill_list"
	CmdNameSkillGet    = "command_skill_get"

	// Agent management commands
	CmdNameAgent       = "command_agent"
	CmdNameAgentCreate = "command_agent_create"
	CmdNameAgentUpdate = "command_agent_update"
	CmdNameAgentDelete = "command_agent_delete"
	CmdNameAgentReload = "command_agent_reload"
	CmdNameAgentList   = "command_agent_list"

	// MCP commands
	CmdNameMcpList = "command_mcp_list"
	CmdPrefixMcp   = "command_mcp"

	// Agent delegation
	CmdNameDelegateToAgent = "command_delegate_to_agent"

	// Info commands
	CmdNameTools   = "command_tools"
	CmdNameSkills  = "command_skills"
	CmdNameStatus  = "command_status"
	CmdNameHistory = "command_history"
	CmdNameModel   = "command_model"
	CmdNameRole    = "command_role"
)

// Command prefix constants for prefix matching
const (
	CmdPrefixSession      = "command_session"
	CmdPrefixProfile      = "command_profile"
	CmdPrefixConfig       = "command_config"
	CmdPrefixKnowledge    = "command_knowledge"
	CmdPrefixSubscription = "command_subscription"
	CmdPrefixSkill        = "command_skill"
	CmdPrefixAgent        = "command_agent"
)
