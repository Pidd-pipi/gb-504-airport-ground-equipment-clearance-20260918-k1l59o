import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable, map } from 'rxjs';
import { ApiResponse, AuditLog, PageResult } from '../types';
import { API_BASE, extractData } from '../utils/request';

export function auditListApi(http: HttpClient, page: number, pageSize: number, entityType = ''): Observable<PageResult<AuditLog>> {
  let params = new HttpParams().set('page', page).set('page_size', pageSize);
  if (entityType) params = params.set('entity_type', entityType);
  return http.get<ApiResponse<PageResult<AuditLog>>>(`${API_BASE}/v1/audit`, { params }).pipe(map(extractData));
}
