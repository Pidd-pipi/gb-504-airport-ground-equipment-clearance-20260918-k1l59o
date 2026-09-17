import { ClearanceState, RiskLevel, UnitState } from '../types';

export const ROLE = {
  ADMIN: 'admin',
  SAFETY_MANAGER: 'safety_manager',
  INSPECTOR: 'inspector',
  WORKER: 'worker',
} as const;

export const ROLE_TEXT: Record<string, string> = {
  admin: '系统管理员',
  safety_manager: '安全放行员',
  inspector: '机坪检查员',
  worker: '地勤操作员',
};

export const UNIT_STATE: Record<string, UnitState> = {
  AVAILABLE: 'available',
  INSPECTION: 'inspection',
  BLOCKED: 'blocked',
  RETIRED: 'retired',
};

export const UNIT_STATE_TEXT: Record<UnitState, string> = {
  available: '可用',
  inspection: '检查中',
  blocked: '已锁定',
  retired: '已退役',
};

export const CLEARANCE_STATE_TEXT: Record<ClearanceState, string> = {
  pending: '待决定',
  cleared: '已放行',
  restricted: '限制放行',
  revoked: '已撤销',
};

export const RISK_TEXT: Record<RiskLevel, string> = {
  low: '低',
  medium: '中',
  high: '高',
  critical: '严重',
};

export const STATUS_TEXT: Record<string, string> = {
  open: '已建立',
  checking: '检查中',
  decisioned: '已决定',
  completed: '已完成',
  pending: '待处理',
  passed: '通过',
  failed: '不通过',
  available: '可用',
  inspection: '检查中',
  blocked: '已锁定',
  retired: '已退役',
  cleared: '已放行',
  restricted: '限制放行',
  revoked: '已撤销',
};
