import { HttpClient } from '@angular/common/http';
import { Injectable, signal } from '@angular/core';
import { forkJoin } from 'rxjs';
import { clearanceListApi, clearanceSummaryApi } from '../api/clearance.api';
import { ClearanceDecision, ClearanceSummary } from '../types';
import { parseHttpError } from '../utils/request';

@Injectable({ providedIn: 'root' })
export class ClearanceStore {
  readonly items = signal<ClearanceDecision[]>([]);
  readonly total = signal(0);
  readonly loading = signal(false);
  readonly error = signal('');
  readonly summary = signal<ClearanceSummary>({ total: 0, states: { pending: 0, cleared: 0, restricted: 0, revoked: 0 }, changed_today: 0 });
  private requestVersion = 0;

  constructor(private readonly http: HttpClient) {}

  load(page = 1, pageSize = 20, state = ''): void {
    const requestVersion = ++this.requestVersion;
    this.loading.set(true);
    this.error.set('');
    forkJoin({ list: clearanceListApi(this.http, page, pageSize, state), summary: clearanceSummaryApi(this.http) }).subscribe({
      next: result => { if (requestVersion !== this.requestVersion) return; this.items.set(result.list.list); this.total.set(result.list.total); this.summary.set(result.summary); this.loading.set(false); },
      error: error => { if (requestVersion !== this.requestVersion) return; this.error.set(parseHttpError(error)); this.loading.set(false); },
    });
  }
}
