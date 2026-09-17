import { HttpClient } from '@angular/common/http';
import { Injectable, signal } from '@angular/core';
import { auditListApi } from '../api/audit.api';
import { AuditLog } from '../types';
import { parseHttpError } from '../utils/request';

@Injectable({ providedIn: 'root' })
export class AuditStore {
  readonly items = signal<AuditLog[]>([]);
  readonly total = signal(0);
  readonly loading = signal(false);
  readonly error = signal('');

  constructor(private readonly http: HttpClient) {}

  load(page = 1, pageSize = 50, entityType = ''): void {
    this.loading.set(true);
    this.error.set('');
    auditListApi(this.http, page, pageSize, entityType).subscribe({
      next: result => { this.items.set(result.list); this.total.set(result.total); this.loading.set(false); },
      error: error => { this.error.set(parseHttpError(error)); this.loading.set(false); },
    });
  }
}
