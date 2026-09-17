import { HttpClient } from '@angular/common/http';
import { Injectable, signal } from '@angular/core';
import { forkJoin } from 'rxjs';
import { checkListApi, checkSummaryApi } from '../api/check.api';
import { SafetyCheck, SafetyCheckSummary } from '../types';
import { parseHttpError } from '../utils/request';

@Injectable({ providedIn: 'root' })
export class CheckStore {
  readonly items = signal<SafetyCheck[]>([]);
  readonly total = signal(0);
  readonly loading = signal(false);
  readonly error = signal('');
  readonly summary = signal<SafetyCheckSummary>({ total: 0, results: { pending: 0, passed: 0, failed: 0 }, critical_pending: 0, completion_percent: 0 });
  private requestVersion = 0;

  constructor(private readonly http: HttpClient) {}

  load(page = 1, pageSize = 20, turnaroundId?: number, result = ''): void {
    const requestVersion = ++this.requestVersion;
    this.loading.set(true);
    this.error.set('');
    forkJoin({ list: checkListApi(this.http, page, pageSize, turnaroundId, result), summary: checkSummaryApi(this.http) }).subscribe({
      next: response => { if (requestVersion !== this.requestVersion) return; this.items.set(response.list.list); this.total.set(response.list.total); this.summary.set(response.summary); this.loading.set(false); },
      error: error => { if (requestVersion !== this.requestVersion) return; this.error.set(parseHttpError(error)); this.loading.set(false); },
    });
  }
}
