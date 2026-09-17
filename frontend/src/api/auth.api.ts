import { HttpClient } from '@angular/common/http';
import { Observable, map } from 'rxjs';
import { ApiResponse, LoginResponse, User } from '../types';
import { API_BASE, extractData } from '../utils/request';

export interface LoginPayload {
  phone: string;
  password: string;
}

export function loginApi(http: HttpClient, payload: LoginPayload): Observable<LoginResponse> {
  return http.post<ApiResponse<LoginResponse>>(`${API_BASE}/v1/auth/login`, payload).pipe(map(extractData));
}

export function meApi(http: HttpClient): Observable<User> {
  return http.get<ApiResponse<User>>(`${API_BASE}/v1/users/me`).pipe(map(extractData));
}
