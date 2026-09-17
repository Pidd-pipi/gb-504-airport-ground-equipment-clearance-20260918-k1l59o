import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable, map } from 'rxjs';
import { ApiResponse, PageResult, SafetyCheck, SafetyCheckSummary } from '../types';
import { API_BASE, extractData } from '../utils/request';

export function checkListApi(http: HttpClient, page = 1, pageSize = 100, turnaroundId?: number, result = ''): Observable<PageResult<SafetyCheck>> {
  let params = new HttpParams().set('page', page).set('page_size', pageSize);
  if (turnaroundId) params = params.set('turnaround_id', turnaroundId);
  if (result) params = params.set('result', result);
  return http.get<ApiResponse<PageResult<SafetyCheck>>>(`${API_BASE}/v1/checks`, { params }).pipe(map(extractData));
}

export function checkSummaryApi(http: HttpClient): Observable<SafetyCheckSummary> {
  return http.get<ApiResponse<SafetyCheckSummary>>(`${API_BASE}/v1/checks/summary`).pipe(map(extractData));
}

export function checkReviewApi(http: HttpClient, id: number, result: 'passed' | 'failed', evidence: string[], remark: string): Observable<SafetyCheck> {
  return http.patch<ApiResponse<SafetyCheck>>(`${API_BASE}/v1/checks/${id}/review`, { result, evidence, remark }).pipe(map(extractData));
}
