import { CommonModule } from '@angular/common';
import { Component, inject, OnInit } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatPaginatorModule, PageEvent } from '@angular/material/paginator';
import { MatSelectModule } from '@angular/material/select';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { MatTableModule } from '@angular/material/table';
import { turnaroundCreateApi, turnaroundStatusApi } from '../api/turnaround.api';
import { RiskBadgeComponent } from '../components/common/risk-badge.component';
import { StatusBadgeComponent } from '../components/common/status-badge.component';
import { ROLE } from '../constants/enums';
import { useAuth } from '../hooks/use-auth';
import { usePagination } from '../hooks/use-pagination';
import { TurnaroundStore } from '../stores/turnaround.store';
import { RiskLevel, Turnaround } from '../types';
import { parseHttpError, useHttp } from '../utils/request';

@Component({
  selector: 'app-turnarounds-page',
  standalone: true,
  imports: [
    CommonModule, ReactiveFormsModule, MatButtonModule, MatFormFieldModule, MatIconModule, MatInputModule,
    MatProgressBarModule, MatPaginatorModule, MatSelectModule, MatSnackBarModule, MatTableModule, RiskBadgeComponent, StatusBadgeComponent,
  ],
  template: `
    <header class="page-head">
      <div><p>TURNAROUND CONTROL</p><h1>航班周转看板</h1><span>按机位跟踪设备投入、检查进度和放行状态</span></div>
      <button *ngIf="canManage" mat-flat-button (click)="showCreate = !showCreate"><mat-icon>{{ showCreate ? 'close' : 'add' }}</mat-icon>{{ showCreate ? '收起' : '建立周转' }}</button>
    </header>

    <section class="metrics">
      <div><span>当前周转</span><strong>{{ store.summary().total }}</strong><small>个作业阶段</small></div>
      <div><span>高风险</span><strong class="warn">{{ store.summary().risks.high + store.summary().risks.critical }}</strong><small>需重点关注</small></div>
      <div><span>检查中</span><strong>{{ store.summary().statuses.checking }}</strong><small>等待现场证据</small></div>
      <div><span>已形成决定</span><strong class="good">{{ store.summary().statuses.decisioned + store.summary().statuses.completed }}</strong><small>可追溯记录</small></div>
    </section>

    <section *ngIf="showCreate" class="create-band">
      <div class="band-title"><mat-icon>add_circle</mat-icon><span><strong>建立航班周转阶段</strong><small>同步创建首个安全检查项和待放行记录</small></span></div>
      <form [formGroup]="form" (ngSubmit)="create()">
        <mat-form-field appearance="outline"><mat-label>航班号</mat-label><input matInput formControlName="flight_no"></mat-form-field>
        <mat-form-field appearance="outline"><mat-label>机位</mat-label><input matInput formControlName="stand"></mat-form-field>
        <mat-form-field appearance="outline"><mat-label>周转阶段</mat-label><mat-select formControlName="phase"><mat-option value="arrival">进港</mat-option><mat-option value="servicing">保障中</mat-option><mat-option value="departure">离港</mat-option></mat-select></mat-form-field>
        <mat-form-field appearance="outline"><mat-label>计划时间</mat-label><input matInput type="datetime-local" formControlName="scheduled_at"></mat-form-field>
        <mat-form-field appearance="outline"><mat-label>风险等级</mat-label><mat-select formControlName="risk_level"><mat-option value="low">低</mat-option><mat-option value="medium">中</mat-option><mat-option value="high">高</mat-option><mat-option value="critical">严重</mat-option></mat-select></mat-form-field>
        <mat-form-field appearance="outline"><mat-label>设备 ID</mat-label><input matInput formControlName="ground_unit_ids" placeholder="1,2"></mat-form-field>
        <mat-form-field appearance="outline"><mat-label>负责人 ID</mat-label><input matInput type="number" formControlName="coordinator_id"></mat-form-field>
        <mat-form-field appearance="outline"><mat-label>首检编码</mat-label><input matInput formControlName="check_code"></mat-form-field>
        <mat-form-field appearance="outline" class="wide"><mat-label>首个检查项</mat-label><input matInput formControlName="check_name"></mat-form-field>
        <div class="form-actions"><button mat-button type="button" (click)="showCreate = false">取消</button><button mat-flat-button type="submit" [disabled]="form.invalid || saving">确认建立</button></div>
      </form>
    </section>

    <section class="table-section">
      <div class="table-tools">
        <div class="segmented">
          <button [class.selected]="statusFilter === ''" (click)="filterStatus('')">全部</button>
          <button [class.selected]="statusFilter === 'checking'" (click)="filterStatus('checking')">检查中</button>
          <button [class.selected]="statusFilter === 'decisioned'" (click)="filterStatus('decisioned')">已决定</button>
        </div>
        <mat-form-field appearance="outline" subscriptSizing="dynamic"><mat-icon matPrefix>search</mat-icon><input matInput placeholder="航班号或机位" #search (keyup.enter)="searchText = search.value; resetAndLoad()"></mat-form-field>
      </div>
      <mat-progress-bar *ngIf="store.loading()" mode="indeterminate"></mat-progress-bar>
      <p class="error" *ngIf="store.error()">{{ store.error() }}</p>
      <div class="table-wrap">
        <table mat-table [dataSource]="store.items()">
          <ng-container matColumnDef="flight"><th mat-header-cell *matHeaderCellDef>航班 / 机位</th><td mat-cell *matCellDef="let row"><strong>{{ row.flight_no }}</strong><small>{{ row.stand }} · {{ phaseText(row.phase) }}</small></td></ng-container>
          <ng-container matColumnDef="schedule"><th mat-header-cell *matHeaderCellDef>计划时间</th><td mat-cell *matCellDef="let row">{{ row.scheduled_at | date:'MM-dd HH:mm' }}</td></ng-container>
          <ng-container matColumnDef="units"><th mat-header-cell *matHeaderCellDef>投入设备</th><td mat-cell *matCellDef="let row">{{ row.ground_unit_ids.length ? row.ground_unit_ids.join(', ') : '未分配' }}</td></ng-container>
          <ng-container matColumnDef="risk"><th mat-header-cell *matHeaderCellDef>风险</th><td mat-cell *matCellDef="let row"><app-risk-badge [level]="row.risk_level"></app-risk-badge></td></ng-container>
          <ng-container matColumnDef="status"><th mat-header-cell *matHeaderCellDef>状态</th><td mat-cell *matCellDef="let row"><app-status-badge [value]="row.status"></app-status-badge></td></ng-container>
          <ng-container matColumnDef="actions"><th mat-header-cell *matHeaderCellDef></th><td mat-cell *matCellDef="let row"><button *ngIf="canManage && row.status === 'open'" mat-stroked-button (click)="startChecks(row)">开始检查</button></td></ng-container>
          <tr mat-header-row *matHeaderRowDef="columns"></tr><tr mat-row *matRowDef="let row; columns: columns"></tr>
        </table>
        <div class="empty" *ngIf="!store.loading() && !store.items().length"><mat-icon>flight</mat-icon><strong>暂无匹配周转</strong><span>调整筛选条件或建立新周转</span></div>
      </div>
      <mat-paginator [length]="store.total()" [pageIndex]="pagination.page() - 1" [pageSize]="pagination.pageSize()" [pageSizeOptions]="[10, 20, 50, 100]" (page)="pageChanged($event)"></mat-paginator>
    </section>
  `,
})
export class TurnaroundsPage implements OnInit {
  readonly store = inject(TurnaroundStore);
  private readonly fb = inject(FormBuilder);
  private readonly http = useHttp();
  private readonly snack = inject(MatSnackBar);
  readonly auth = useAuth();
  readonly pagination = usePagination(20);
  readonly canManage = this.auth.hasRole(ROLE.ADMIN, ROLE.SAFETY_MANAGER);
  readonly columns = ['flight', 'schedule', 'units', 'risk', 'status', 'actions'];
  statusFilter = '';
  searchText = '';
  showCreate = false;
  saving = false;
  readonly form = this.fb.nonNullable.group({
    flight_no: ['CA', Validators.required], stand: ['A12', Validators.required],
    phase: ['servicing' as 'arrival' | 'servicing' | 'departure', Validators.required],
    scheduled_at: [this.localDateTime(), Validators.required], risk_level: ['medium' as RiskLevel, Validators.required],
    ground_unit_ids: ['1'], coordinator_id: [2, [Validators.required, Validators.min(1)]],
    check_code: ['OPS-001', Validators.required], check_name: ['设备外观、制动与安全区域确认', Validators.required],
  });

  ngOnInit(): void { this.reload(); }
  phaseText(phase: string): string { return ({ arrival: '进港', servicing: '保障中', departure: '离港' } as Record<string, string>)[phase] || phase; }
  filterStatus(status: string): void { this.statusFilter = status; this.resetAndLoad(); }
  reload(): void { this.store.load(this.pagination.page(), this.pagination.pageSize(), this.statusFilter, '', this.searchText); }
  resetAndLoad(): void { this.pagination.reset(); this.reload(); }
  pageChanged(event: PageEvent): void { this.pagination.setPage(event.pageIndex + 1); this.pagination.pageSize.set(event.pageSize); this.reload(); }

  create(): void {
    if (this.form.invalid) return;
    this.saving = true;
    const value = this.form.getRawValue();
    const unitIds = value.ground_unit_ids.split(',').map(item => item.trim()).filter(Boolean);
    turnaroundCreateApi(this.http, {
      flight_no: value.flight_no.trim().toUpperCase(), stand: value.stand.trim().toUpperCase(), phase: value.phase,
      scheduled_at: new Date(value.scheduled_at).toISOString(), risk_level: value.risk_level,
      ground_unit_ids: unitIds, coordinator_id: value.coordinator_id,
      checks: [{ ground_unit_id: unitIds[0] ? Number(unitIds[0]) : null, check_code: value.check_code.trim(), item_name: value.check_name.trim(), risk_level: value.risk_level, evidence: [] }],
    }).subscribe({
      next: () => { this.saving = false; this.showCreate = false; this.reload(); this.snack.open('周转阶段已建立', '关闭', { duration: 2500 }); },
      error: error => { this.saving = false; this.snack.open(parseHttpError(error), '关闭', { duration: 4000 }); },
    });
  }

  startChecks(row: Turnaround): void {
    turnaroundStatusApi(this.http, row.id, 'checking', row.version).subscribe({
      next: () => { this.reload(); this.snack.open('周转已进入检查阶段', '关闭', { duration: 2200 }); },
      error: error => this.snack.open(parseHttpError(error), '关闭', { duration: 4000 }),
    });
  }

  private localDateTime(): string {
    const date = new Date(Date.now() + 60 * 60 * 1000);
    date.setMinutes(date.getMinutes() - date.getTimezoneOffset());
    return date.toISOString().slice(0, 16);
  }
}
