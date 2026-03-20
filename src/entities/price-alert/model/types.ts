export interface PriceAlert {
  id: string;
  case_id: string;
  pipeline_id?: string;
  condition: string;
  label: string;
  triggered: boolean;
  triggered_at?: string;
  created_at: string;
}
