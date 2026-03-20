export interface AgentBlock {
  id: string;
  user_id: string;
  name: string;
  instruction: string;
  system_prompt?: string;
  allowed_tools?: string[];
  output_schema?: Record<string, unknown>;
  is_public: boolean;
  created_at: string;
}
