import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable, map } from 'rxjs';
import { ApiResponse, PageResult, RiskLevel, Turnaround, TurnaroundSummary } from '../types';
import { API_BASE, extractData } from '../utils/request';

export interface TurnaroundPayload {
  flight_no: string;
  stand: string;
  phase: 'arrival' | 'servicing' | 'departure';
  scheduled_at: string;
  risk_level: RiskLevel;
  ground_unit_ids: string[];
  coordinator_id: number;
  checks: Array<{
    ground_unit_id: number | null;
    check_code: string;
    item_name: string;
    risk_level: RiskLevel;
    evidence: string[];
  }>;
}

export function turnaroundListApi(http: HttpClient, page = 1, pageSize = 50, status = '', risk = '', search = ''): Observable<PageResult<Turnaround>> {
  let params = new HttpParams().set('page', page).set('page_size', pageSize);
  if (status) params = params.set('status', status);
  if (risk) params = params.set('risk_level', risk);
  if (search) params = params.set('search', search);
  return http.get<ApiResponse<PageResult<Turnaround>>>(`${API_BASE}/v1/turnarounds`, { params }).pipe(map(extractData));
}

export function turnaroundSummaryApi(http: HttpClient): Observable<TurnaroundSummary> {
  return http.get<ApiResponse<TurnaroundSummary>>(`${API_BASE}/v1/turnarounds/summary`).pipe(map(extractData));
}

export function turnaroundCreateApi(http: HttpClient, payload: TurnaroundPayload): Observable<Turnaround> {
  return http.post<ApiResponse<Turnaround>>(`${API_BASE}/v1/turnarounds`, payload).pipe(map(extractData));
}

export function turnaroundStatusApi(http: HttpClient, id: number, status: string, version: number): Observable<Turnaround> {
  return http.patch<ApiResponse<Turnaround>>(`${API_BASE}/v1/turnarounds/${id}/status`, { status, version }).pipe(map(extractData));
}
