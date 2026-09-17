import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable, map } from 'rxjs';
import { ApiResponse, GroundUnit, GroundUnitSummary, PageResult, UnitState } from '../types';
import { API_BASE, extractData } from '../utils/request';

export interface GroundUnitPayload {
  unit_code: string;
  name: string;
  unit_type: string;
  stand: string;
  state: UnitState;
  notes: string;
}

export function groundUnitListApi(http: HttpClient, page = 1, pageSize = 50, state = '', search = ''): Observable<PageResult<GroundUnit>> {
  let params = new HttpParams().set('page', page).set('page_size', pageSize);
  if (state) params = params.set('state', state);
  if (search) params = params.set('search', search);
  return http.get<ApiResponse<PageResult<GroundUnit>>>(`${API_BASE}/v1/ground-units`, { params }).pipe(map(extractData));
}

export function groundUnitSummaryApi(http: HttpClient): Observable<GroundUnitSummary> {
  return http.get<ApiResponse<GroundUnitSummary>>(`${API_BASE}/v1/ground-units/summary`).pipe(map(extractData));
}

export function groundUnitCreateApi(http: HttpClient, payload: GroundUnitPayload): Observable<GroundUnit> {
  return http.post<ApiResponse<GroundUnit>>(`${API_BASE}/v1/ground-units`, payload).pipe(map(extractData));
}

export function groundUnitStateApi(http: HttpClient, id: number, state: UnitState, notes: string, version: number): Observable<GroundUnit> {
  return http.patch<ApiResponse<GroundUnit>>(`${API_BASE}/v1/ground-units/${id}/state`, { state, notes, version }).pipe(map(extractData));
}
