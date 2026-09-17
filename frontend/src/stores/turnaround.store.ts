import { HttpClient } from '@angular/common/http';
import { Injectable, signal } from '@angular/core';
import { forkJoin } from 'rxjs';
import { turnaroundListApi, turnaroundSummaryApi } from '../api/turnaround.api';
import { Turnaround, TurnaroundSummary } from '../types';
import { parseHttpError } from '../utils/request';

@Injectable({ providedIn: 'root' })
export class TurnaroundStore {
  readonly items = signal<Turnaround[]>([]);
  readonly total = signal(0);
  readonly loading = signal(false);
  readonly error = signal('');
  readonly summary = signal<TurnaroundSummary>({ total: 0, statuses: { open: 0, checking: 0, decisioned: 0, completed: 0 }, risks: { low: 0, medium: 0, high: 0, critical: 0 }, due_within_two_hours: 0 });
  private requestVersion = 0;

  constructor(private readonly http: HttpClient) {}

  load(page = 1, pageSize = 20, status = '', risk = '', search = ''): void {
    const requestVersion = ++this.requestVersion;
    this.loading.set(true);
    this.error.set('');
    forkJoin({ list: turnaroundListApi(this.http, page, pageSize, status, risk, search), summary: turnaroundSummaryApi(this.http) }).subscribe({
      next: result => { if (requestVersion !== this.requestVersion) return; this.items.set(result.list.list); this.total.set(result.list.total); this.summary.set(result.summary); this.loading.set(false); },
      error: error => { if (requestVersion !== this.requestVersion) return; this.error.set(parseHttpError(error)); this.loading.set(false); },
    });
  }
}
