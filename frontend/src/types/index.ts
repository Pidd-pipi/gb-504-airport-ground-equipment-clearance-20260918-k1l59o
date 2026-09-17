export interface ApiResponse<T = unknown> {
  code: number;
  message: string;
  data: T;
}

export interface PageResult<T> {
  list: T[];
  total: number;
  page: number;
  page_size: number;
}

export interface User {
  id: number;
  phone: string;
  name: string;
  role: string;
  created_at: string;
}

export interface LoginResponse {
  token: string;
  user: User;
}

export type UnitState = 'available' | 'inspection' | 'blocked' | 'retired';
export type ClearanceState = 'pending' | 'cleared' | 'restricted' | 'revoked';
export type RiskLevel = 'low' | 'medium' | 'high' | 'critical';
export type CheckResult = 'pending' | 'passed' | 'failed';

export interface GroundUnit {
  id: number;
  unit_code: string;
  name: string;
  unit_type: string;
  stand: string;
  state: UnitState;
  last_inspection_at: string | null;
  notes: string;
  version: number;
  created_at: string;
  updated_at: string;
}

export interface GroundUnitSummary {
  total: number;
  states: Record<UnitState, number>;
  types: Record<string, number>;
  dispatchable: number;
  unavailable: number;
}

export interface Turnaround {
  id: number;
  flight_no: string;
  stand: string;
  phase: 'arrival' | 'servicing' | 'departure';
  scheduled_at: string;
  risk_level: RiskLevel;
  status: 'open' | 'checking' | 'decisioned' | 'completed';
  ground_unit_ids: string[];
  coordinator_id: number;
  version: number;
  created_at: string;
  updated_at: string;
}

export interface TurnaroundSummary {
  total: number;
  statuses: Record<Turnaround['status'], number>;
  risks: Record<RiskLevel, number>;
  due_within_two_hours: number;
}

export interface SafetyCheck {
  id: number;
  turnaround_id: number;
  ground_unit_id: number | null;
  sequence: number;
  check_code: string;
  item_name: string;
  risk_level: RiskLevel;
  result: CheckResult;
  evidence: string[];
  remark: string;
  checked_by: number;
  checked_at: string | null;
}

export interface SafetyCheckSummary {
  total: number;
  results: Record<CheckResult, number>;
  critical_pending: number;
  completion_percent: number;
}

export interface ClearanceDecision {
  id: number;
  turnaround_id: number;
  state: ClearanceState;
  previous_state: ClearanceState | '';
  restrictions: string;
  reason: string;
  evidence: string[];
  operator_id: number;
  request_id: string;
  decided_at: string;
  created_at: string;
  updated_at: string;
}

export interface ClearanceSummary {
  total: number;
  states: Record<ClearanceState, number>;
  changed_today: number;
}

export interface AuditLog {
  id: number;
  operator_id: number;
  operator_name: string;
  action: string;
  entity_type: string;
  entity_id: string;
  detail: string;
  ip: string;
  created_at: string;
}
