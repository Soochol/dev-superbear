export interface PriceAlert {
  id: string;
  case_id: string;
  pipeline_id: string | null;
  condition: string;
  label: string;
  triggered: boolean;
  triggered_at: string | null;
  created_at: string;
}

export interface AlertsResponse {
  pending: PriceAlert[];
  triggered: PriceAlert[];
}
