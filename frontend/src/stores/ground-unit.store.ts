import { HttpClient } from '@angular/common/http';
import { Injectable, signal } from '@angular/core';
import { forkJoin } from 'rxjs';
import { groundUnitListApi, groundUnitSummaryApi } from '../api/ground-unit.api';
import { GroundUnit, GroundUnitSummary } from '../types';
import { parseHttpError } from '../utils/request';

@Injectable({ providedIn: 'root' })
export class GroundUnitStore {
  readonly items = signal<GroundUnit[]>([]);
  readonly total = signal(0);
  readonly loading = signal(false);
  readonly error = signal('');
  readonly summary = signal<GroundUnitSummary>({ total: 0, states: { available: 0, inspection: 0, blocked: 0, retired: 0 }, types: {}, dispatchable: 0, unavailable: 0 });
  private requestVersion = 0;

  constructor(private readonly http: HttpClient) {}

  load(page = 1, pageSize = 20, state = '', search = ''): void {
    const requestVersion = ++this.requestVersion;
    this.loading.set(true);
    this.error.set('');
    forkJoin({ list: groundUnitListApi(this.http, page, pageSize, state, search), summary: groundUnitSummaryApi(this.http) }).subscribe({
      next: result => { if (requestVersion !== this.requestVersion) return; this.items.set(result.list.list); this.total.set(result.list.total); this.summary.set(result.summary); this.loading.set(false); },
      error: error => { if (requestVersion !== this.requestVersion) return; this.error.set(parseHttpError(error)); this.loading.set(false); },
    });
  }
}
