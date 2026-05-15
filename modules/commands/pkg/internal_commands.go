// Package commands provides internal command definitions and metadata.
// This package is UI-agnostic and can be used across TUI, CLI, and adapter scenarios.
package commands

import "github.com/oneliang/aura/shared/pkg/i18n"

// GetInternalCommands returns all internal commands with their descriptions.
// The returned descriptions are translated using i18n based on the current locale.
func GetInternalCommands() []CommandInfo {
	return []CommandInfo{
		// Basic commands
		{
			Name:        CmdNameExit,
			DisplayName: i18n.T("internal_command.exit.name"),
			Description: i18n.T("internal_command.exit.desc"),
		},
		{
			Name:        CmdNameQuit,
			DisplayName: i18n.T("internal_command.quit.name"),
			Description: i18n.T("internal_command.quit.desc"),
		},
		{
			Name:        CmdNameClear,
			DisplayName: i18n.T("internal_command.clear.name"),
			Description: i18n.T("internal_command.clear.desc"),
		},
		{
			Name:        CmdNameCompact,
			DisplayName: i18n.T("internal_command.compact.name"),
			Description: i18n.T("internal_command.compact.desc"),
		},
		{
			Name:        CmdNameHelp,
			DisplayName: i18n.T("internal_command.help.name"),
			Description: i18n.T("internal_command.help.desc"),
		},
		{
			Name:        CmdNameMemory,
			DisplayName: i18n.T("internal_command.memory.name"),
			Description: i18n.T("internal_command.memory.desc"),
		},

		// Session commands
		{
			Name:        CmdNameSessions,
			DisplayName: i18n.T("internal_command.sessions.name"),
			Description: i18n.T("internal_command.sessions.desc"),
		},
		{
			Name:        CmdNameSessionCreate,
			DisplayName: i18n.T("internal_command.session_create.name"),
			Description: i18n.T("internal_command.session_create.desc"),
			Params: []ParamInfo{
				{Name: "name", Type: "string", Required: false, Desc: "Session name"},
				{Name: "role", Type: "string", Required: false, Desc: "Session role"},
			},
		},
		{
			Name:        CmdNameSessionShow,
			DisplayName: i18n.T("internal_command.session_show.name"),
			Description: i18n.T("internal_command.session_show.desc"),
			Params: []ParamInfo{
				{Name: "id", Type: "string", Required: true, Desc: "Session ID"},
			},
		},
		{
			Name:        CmdNameSessionDelete,
			DisplayName: i18n.T("internal_command.session_delete.name"),
			Description: i18n.T("internal_command.session_delete.desc"),
			Params: []ParamInfo{
				{Name: "id", Type: "string", Required: true, Desc: "Session ID to delete"},
			},
		},
		{
			Name:        CmdNameSessionUpdate,
			DisplayName: i18n.T("internal_command.session_update.name"),
			Description: i18n.T("internal_command.session_update.desc"),
			Params: []ParamInfo{
				{Name: "id", Type: "string", Required: true, Desc: "Session ID"},
				{Name: "role", Type: "string", Required: false, Desc: "Role name"},
			},
		},

		// Profile commands
		{
			Name:        CmdNameProfile,
			DisplayName: i18n.T("internal_command.profile.name"),
			Description: i18n.T("internal_command.profile.desc"),
		},
		{
			Name:        CmdNameProfileShow,
			DisplayName: i18n.T("internal_command.profile_show.name"),
			Description: i18n.T("internal_command.profile_show.desc"),
		},
		{
			Name:        CmdNameProfileImport,
			DisplayName: i18n.T("internal_command.profile_import.name"),
			Description: i18n.T("internal_command.profile_import.desc"),
			Params: []ParamInfo{
				{Name: "path", Type: "string", Required: true, Desc: "Profile file path"},
			},
		},

		// Config commands
		{
			Name:        CmdNameConfig,
			DisplayName: i18n.T("internal_command.config.name"),
			Description: i18n.T("internal_command.config.desc"),
		},
		{
			Name:        CmdNameConfigShow,
			DisplayName: i18n.T("internal_command.config_show.name"),
			Description: i18n.T("internal_command.config_show.desc"),
		},
		{
			Name:        CmdNameConfigPath,
			DisplayName: i18n.T("internal_command.config_path.name"),
			Description: i18n.T("internal_command.config_path.desc"),
		},
		{
			Name:        CmdNameConfigGet,
			DisplayName: i18n.T("internal_command.config_get.name"),
			Description: i18n.T("internal_command.config_get.desc"),
			Params: []ParamInfo{
				{Name: "key", Type: "string", Required: true, Desc: "Config key in dot notation (e.g., llm.model, agent.planning_mode)"},
			},
		},

		// Knowledge commands
		{
			Name:        CmdNameKnowledge,
			DisplayName: i18n.T("internal_command.knowledge.name"),
			Description: i18n.T("internal_command.knowledge.desc"),
		},
		{
			Name:        CmdNameKnowledgeSearch,
			DisplayName: i18n.T("internal_command.knowledge_search.name"),
			Description: i18n.T("internal_command.knowledge_search.desc"),
			Params: []ParamInfo{
				{Name: "query", Type: "string", Required: true, Desc: "Search query"},
			},
		},
		{
			Name:        CmdNameKnowledgeImport,
			DisplayName: i18n.T("internal_command.knowledge_import.name"),
			Description: i18n.T("internal_command.knowledge_import.desc"),
			Params: []ParamInfo{
				{Name: "path", Type: "string", Required: true, Desc: "File or directory path to import"},
			},
		},
		{
			Name:        CmdNameKnowledgeStats,
			DisplayName: i18n.T("internal_command.knowledge_stats.name"),
			Description: i18n.T("internal_command.knowledge_stats.desc"),
		},

		// Subscription commands
		{
			Name:        CmdNameSubscription,
			DisplayName: i18n.T("internal_command.subscription.name"),
			Description: i18n.T("internal_command.subscription.desc"),
		},
		{
			Name:        CmdNameSubscriptionShow,
			DisplayName: i18n.T("internal_command.subscription_show.name"),
			Description: i18n.T("internal_command.subscription_show.desc"),
			Params: []ParamInfo{
				{Name: "session_id", Type: "string", Required: false, Desc: "Session ID (empty for all sessions)"},
			},
		},
		{
			Name:        CmdNameSubscriptionAdd,
			DisplayName: i18n.T("internal_command.subscription_add.name"),
			Description: i18n.T("internal_command.subscription_add.desc"),
			Params: []ParamInfo{
				{Name: "session_id", Type: "string", Required: true, Desc: "Session ID"},
				{Name: "trigger", Type: "string", Required: true, Desc: "Trigger keyword"},
				{Name: "source", Type: "string", Required: false, Desc: "Trigger source (feishu/email/cron/api/*)"},
			},
		},
		{
			Name:        CmdNameSubscriptionDelete,
			DisplayName: i18n.T("internal_command.subscription_delete.name"),
			Description: i18n.T("internal_command.subscription_delete.desc"),
			Params: []ParamInfo{
				{Name: "session_id", Type: "string", Required: true, Desc: "Session ID"},
				{Name: "subscription_id", Type: "string", Required: true, Desc: "Subscription ID to delete"},
			},
		},

		// Agent delegation command
		{
			Name:        CmdNameDelegateToAgent,
			DisplayName: "Delegate to Agent",
			Description: "Delegate a task to a specialized SubAgent",
			Params: []ParamInfo{
				{Name: "agent", Type: "string", Required: true, Desc: "Agent name to delegate to"},
				{Name: "task", Type: "string", Required: true, Desc: "Task description to delegate"},
			},
		},

		// Skill management commands
		{
			Name:        CmdNameSkillCreate,
			DisplayName: i18n.T("internal_command.skill_create.name"),
			Description: i18n.T("internal_command.skill_create.desc"),
			Params: []ParamInfo{
				{Name: "name", Type: "string", Required: true, Desc: "Skill name (unique identifier)"},
				{Name: "description", Type: "string", Required: true, Desc: "Skill description (triggers LLM to use this skill)"},
				{Name: "body", Type: "string", Required: true, Desc: "Skill body (Markdown instructions)"},
			},
		},
		{
			Name:        CmdNameSkillUpdate,
			DisplayName: i18n.T("internal_command.skill_update.name"),
			Description: i18n.T("internal_command.skill_update.desc"),
			Params: []ParamInfo{
				{Name: "name", Type: "string", Required: true, Desc: "Skill name to update"},
				{Name: "description", Type: "string", Required: false, Desc: "New description"},
				{Name: "body", Type: "string", Required: false, Desc: "New body content"},
			},
		},
		{
			Name:        CmdNameSkillDelete,
			DisplayName: i18n.T("internal_command.skill_delete.name"),
			Description: i18n.T("internal_command.skill_delete.desc"),
			Params: []ParamInfo{
				{Name: "name", Type: "string", Required: true, Desc: "Skill name to delete"},
			},
		},
		{
			Name:        CmdNameSkillReload,
			DisplayName: i18n.T("internal_command.skill_reload.name"),
			Description: i18n.T("internal_command.skill_reload.desc"),
		},
		{
			Name:        CmdNameSkillList,
			DisplayName: i18n.T("internal_command.skill_list.name"),
			Description: i18n.T("internal_command.skill_list.desc"),
		},
		{
			Name:        CmdNameSkillGet,
			DisplayName: i18n.T("internal_command.skill_get.name"),
			Description: i18n.T("internal_command.skill_get.desc"),
			Params: []ParamInfo{
				{Name: "name", Type: "string", Required: true, Desc: "Skill name"},
			},
		},

		// Agent management commands
		{
			Name:        CmdNameAgentCreate,
			DisplayName: i18n.T("internal_command.agent_create.name"),
			Description: i18n.T("internal_command.agent_create.desc"),
			Params: []ParamInfo{
				{Name: "name", Type: "string", Required: true, Desc: "Agent name (unique identifier)"},
				{Name: "description", Type: "string", Required: true, Desc: "Agent description (triggers LLM to delegate)"},
				{Name: "body", Type: "string", Required: true, Desc: "Agent body (Markdown system prompt)"},
				{Name: "llm_model", Type: "string", Required: false, Desc: "LLM model override"},
				{Name: "temperature", Type: "float", Required: false, Desc: "LLM temperature (0.0-1.0)"},
			},
		},
		{
			Name:        CmdNameAgentUpdate,
			DisplayName: i18n.T("internal_command.agent_update.name"),
			Description: i18n.T("internal_command.agent_update.desc"),
			Params: []ParamInfo{
				{Name: "name", Type: "string", Required: true, Desc: "Agent name to update"},
				{Name: "description", Type: "string", Required: false, Desc: "New description"},
				{Name: "body", Type: "string", Required: false, Desc: "New body content"},
				{Name: "llm_model", Type: "string", Required: false, Desc: "New LLM model override"},
			},
		},
		{
			Name:        CmdNameAgentDelete,
			DisplayName: i18n.T("internal_command.agent_delete.name"),
			Description: i18n.T("internal_command.agent_delete.desc"),
			Params: []ParamInfo{
				{Name: "name", Type: "string", Required: true, Desc: "Agent name to delete"},
			},
		},
		{
			Name:        CmdNameAgentReload,
			DisplayName: i18n.T("internal_command.agent_reload.name"),
			Description: i18n.T("internal_command.agent_reload.desc"),
		},
		{
			Name:        CmdNameAgentList,
			DisplayName: i18n.T("internal_command.agent_list.name"),
			Description: i18n.T("internal_command.agent_list.desc"),
		},

		// MCP commands
		{
			Name:        CmdNameMcpList,
			DisplayName: i18n.T("internal_command.mcp_list.name"),
			Description: i18n.T("internal_command.mcp_list.desc"),
		},

		// Info commands
		{
			Name:        CmdNameTools,
			DisplayName: i18n.T("internal_command.tools.name"),
			Description: i18n.T("internal_command.tools.desc"),
		},
		{
			Name:        CmdNameSkills,
			DisplayName: i18n.T("internal_command.skills.name"),
			Description: i18n.T("internal_command.skills.desc"),
		},
		{
			Name:        CmdNameStatus,
			DisplayName: i18n.T("internal_command.status.name"),
			Description: i18n.T("internal_command.status.desc"),
		},
		{
			Name:        CmdNameHistory,
			DisplayName: i18n.T("internal_command.history.name"),
			Description: i18n.T("internal_command.history.desc"),
		},
		{
			Name:        CmdNameModel,
			DisplayName: i18n.T("internal_command.model.name"),
			Description: i18n.T("internal_command.model.desc"),
		},
		{
			Name:        CmdNameRole,
			DisplayName: i18n.T("internal_command.role.name"),
			Description: i18n.T("internal_command.role.desc"),
		},
	}
}
