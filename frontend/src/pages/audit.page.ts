import { CommonModule } from '@angular/common';
import { Component, inject, OnInit } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatPaginatorModule, PageEvent } from '@angular/material/paginator';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatSelectModule } from '@angular/material/select';
import { MatTableModule } from '@angular/material/table';
import { usePagination } from '../hooks/use-pagination';
import { AuditStore } from '../stores/audit.store';
import { AuditLog } from '../types';

@Component({
  selector: 'app-audit-page',
  standalone: true,
  imports: [CommonModule, MatButtonModule, MatFormFieldModule, MatIconModule, MatPaginatorModule, MatProgressBarModule, MatSelectModule, MatTableModule],
  template: `
    <header class="page-head">
      <div><p>AUDIT TRAIL</p><h1>审计追踪</h1><span>检索状态变更、操作证据、请求来源和责任人员</span></div>
      <button mat-stroked-button (click)="reload()"><mat-icon>refresh</mat-icon>刷新</button>
    </header>
    <section class="audit-summary"><mat-icon>verified</mat-icon><span><strong>审计链完整</strong>所有写操作由请求追踪中间件自动记录，放行迁移额外保存前后状态和证据。</span><small>{{ store.total() }} 条记录</small></section>
    <section class="table-section">
      <div class="table-tools">
        <mat-form-field appearance="outline" subscriptSizing="dynamic"><mat-label>实体类型</mat-label><mat-select [(value)]="entityType" (selectionChange)="resetAndLoad()"><mat-option value="">全部实体</mat-option><mat-option value="ground-units">地面设备</mat-option><mat-option value="turnarounds">航班周转</mat-option><mat-option value="checks">检查项</mat-option><mat-option value="clearance">安全放行</mat-option></mat-select></mat-form-field>
      </div>
      <mat-progress-bar *ngIf="store.loading()" mode="indeterminate"></mat-progress-bar>
      <p class="error" *ngIf="store.error()">{{ store.error() }}</p>
      <div class="table-wrap">
        <table mat-table [dataSource]="store.items()">
          <ng-container matColumnDef="time"><th mat-header-cell *matHeaderCellDef>时间</th><td mat-cell *matCellDef="let row">{{ row.created_at | date:'MM-dd HH:mm:ss' }}</td></ng-container>
          <ng-container matColumnDef="operator"><th mat-header-cell *matHeaderCellDef>操作人</th><td mat-cell *matCellDef="let row"><strong>{{ row.operator_name || '系统' }}</strong><small>ID {{ row.operator_id || '-' }} · {{ row.ip || '-' }}</small></td></ng-container>
          <ng-container matColumnDef="action"><th mat-header-cell *matHeaderCellDef>动作</th><td mat-cell *matCellDef="let row"><span class="action-code">{{ row.action }}</span></td></ng-container>
          <ng-container matColumnDef="entity"><th mat-header-cell *matHeaderCellDef>对象</th><td mat-cell *matCellDef="let row">{{ entityText(row.entity_type) }}<small>#{{ row.entity_id || '-' }}</small></td></ng-container>
          <ng-container matColumnDef="detail"><th mat-header-cell *matHeaderCellDef>审计内容</th><td mat-cell *matCellDef="let row"><button mat-button class="detail-button" (click)="selected = selected?.id === row.id ? null : row"><mat-icon>{{ selected?.id === row.id ? 'expand_less' : 'expand_more' }}</mat-icon>{{ selected?.id === row.id ? '收起' : '查看' }}</button></td></ng-container>
          <tr mat-header-row *matHeaderRowDef="columns"></tr><tr mat-row *matRowDef="let row; columns: columns"></tr>
        </table>
        <pre *ngIf="selected" class="audit-detail">{{ prettyDetail(selected.detail) }}</pre>
        <div class="empty" *ngIf="!store.loading() && !store.items().length"><mat-icon>history</mat-icon><strong>暂无审计记录</strong><span>完成一次业务写操作后会在此显示</span></div>
      </div>
      <mat-paginator [length]="store.total()" [pageIndex]="pagination.page() - 1" [pageSize]="pagination.pageSize()" [pageSizeOptions]="[20, 50, 100]" (page)="pageChanged($event)"></mat-paginator>
    </section>
  `,
})
export class AuditPage implements OnInit {
  readonly store = inject(AuditStore);
  readonly pagination = usePagination(50);
  readonly columns = ['time', 'operator', 'action', 'entity', 'detail'];
  entityType = '';
  selected: AuditLog | null = null;

  ngOnInit(): void { this.reload(); }
  reload(): void { this.store.load(this.pagination.page(), this.pagination.pageSize(), this.entityType); }
  resetAndLoad(): void { this.pagination.reset(); this.selected = null; this.reload(); }
  pageChanged(event: PageEvent): void {
    this.pagination.setPage(event.pageIndex + 1);
    this.pagination.pageSize.set(event.pageSize);
    this.reload();
  }
  entityText(value: string): string {
    return ({ 'ground-units': '地面设备', turnarounds: '航班周转', checks: '检查项', clearance: '安全放行', users: '用户' } as Record<string, string>)[value] || value;
  }
  prettyDetail(value: string): string {
    try { return JSON.stringify(JSON.parse(value), null, 2); } catch { return value; }
  }
}
