import { CommonModule } from '@angular/common';
import { Component, inject, OnInit } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatDialog, MatDialogModule } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatPaginatorModule, PageEvent } from '@angular/material/paginator';
import { MatSelectModule } from '@angular/material/select';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { checkReviewApi } from '../api/check.api';
import { ClearancePanelComponent } from '../components/common/clearance-panel.component';
import { ConfirmDialogComponent } from '../components/common/confirm-dialog.component';
import { EvidenceListComponent } from '../components/common/evidence-list.component';
import { RiskBadgeComponent } from '../components/common/risk-badge.component';
import { StatusBadgeComponent } from '../components/common/status-badge.component';
import { ROLE } from '../constants/enums';
import { useAuth } from '../hooks/use-auth';
import { usePagination } from '../hooks/use-pagination';
import { CheckStore } from '../stores/check.store';
import { ClearanceStore } from '../stores/clearance.store';
import { SafetyCheck } from '../types';
import { parseHttpError, useHttp } from '../utils/request';

@Component({
  selector: 'app-checks-page',
  standalone: true,
  imports: [
    CommonModule, ReactiveFormsModule, MatButtonModule, MatDialogModule, MatFormFieldModule, MatIconModule,
    MatInputModule, MatPaginatorModule, MatProgressBarModule, MatSelectModule, MatSnackBarModule, ClearancePanelComponent,
    EvidenceListComponent, RiskBadgeComponent, StatusBadgeComponent,
  ],
  template: `
    <header class="page-head">
      <div><p>SAFETY CHECK DESK</p><h1>检查工作台</h1><span>逐项核对设备条件，并将现场证据纳入放行依据</span></div>
      <div class="head-status"><span class="pulse"></span>{{ pendingCount }} 项待检查</div>
    </header>
    <section class="metrics">
      <div><span>检查项</span><strong>{{ store.summary().total }}</strong><small>当前作业清单</small></div>
      <div><span>待处理</span><strong>{{ pendingCount }}</strong><small>需要现场确认</small></div>
      <div><span>已通过</span><strong class="good">{{ store.summary().results.passed }}</strong><small>证据已归档</small></div>
      <div><span>未通过</span><strong class="danger">{{ store.summary().results.failed }}</strong><small>阻断完全放行</small></div>
    </section>

    <section class="check-layout">
      <div class="check-list">
        <div class="table-tools">
          <div class="segmented"><button [class.selected]="filter === ''" (click)="setFilter('')">全部</button><button [class.selected]="filter === 'pending'" (click)="setFilter('pending')">待检查</button><button [class.selected]="filter === 'failed'" (click)="setFilter('failed')">未通过</button></div>
        </div>
        <mat-progress-bar *ngIf="store.loading()" mode="indeterminate"></mat-progress-bar>
        <p class="error" *ngIf="store.error()">{{ store.error() }}</p>
        <button type="button" class="check-row" *ngFor="let item of store.items()" [class.selected]="selected?.id === item.id" (click)="select(item)">
          <span class="sequence">{{ item.sequence | number:'2.0' }}</span>
          <span class="check-copy"><strong>{{ item.item_name }}</strong><small>{{ item.check_code }} · 周转 #{{ item.turnaround_id }}<span *ngIf="item.ground_unit_id"> · 设备 #{{ item.ground_unit_id }}</span></small></span>
          <app-risk-badge [level]="item.risk_level"></app-risk-badge>
          <app-status-badge [value]="item.result"></app-status-badge>
          <mat-icon>chevron_right</mat-icon>
        </button>
        <div class="empty" *ngIf="!store.loading() && !store.items().length"><mat-icon>fact_check</mat-icon><strong>没有匹配检查项</strong><span>当前筛选下已处理完毕</span></div>
        <mat-paginator [length]="store.total()" [pageIndex]="pagination.page() - 1" [pageSize]="pagination.pageSize()" [pageSizeOptions]="[10, 20, 50, 100]" (page)="pageChanged($event)"></mat-paginator>
      </div>

      <aside class="inspection-pane" *ngIf="selected; else selectHint">
        <header><span><small>检查项 {{ selected.check_code }}</small><strong>{{ selected.item_name }}</strong></span><app-status-badge [value]="selected.result"></app-status-badge></header>
        <div class="inspection-meta"><span><small>周转编号</small>#{{ selected.turnaround_id }}</span><span><small>设备编号</small>{{ selected.ground_unit_id ? '#' + selected.ground_unit_id : '通用项' }}</span><span><small>风险级别</small><app-risk-badge [level]="selected.risk_level"></app-risk-badge></span></div>
        <section class="evidence-block"><h3>现场证据</h3><app-evidence-list [items]="selected.evidence || []"></app-evidence-list><p *ngIf="selected.remark">{{ selected.remark }}</p></section>
        <form *ngIf="selected.result === 'pending' && canReview" [formGroup]="reviewForm" (ngSubmit)="review()" class="review-form">
          <h3>形成检查结论</h3>
          <mat-form-field appearance="outline"><mat-label>检查结论</mat-label><mat-select formControlName="result"><mat-option value="passed">检查通过</mat-option><mat-option value="failed">检查不通过</mat-option></mat-select></mat-form-field>
          <mat-form-field appearance="outline"><mat-label>证据文件名 / 编号</mat-label><input matInput formControlName="evidence" placeholder="例如 GPU-test-0822.jpg"></mat-form-field>
          <mat-form-field appearance="outline"><mat-label>检查说明</mat-label><textarea matInput rows="3" formControlName="remark"></textarea></mat-form-field>
          <button mat-flat-button type="submit" [disabled]="reviewForm.invalid || saving"><mat-icon>task_alt</mat-icon>提交结论</button>
        </form>
        <app-clearance-panel [decision]="decisionFor(selected.turnaround_id)" [compact]="true"></app-clearance-panel>
      </aside>
      <ng-template #selectHint><aside class="inspection-pane placeholder"><mat-icon>touch_app</mat-icon><strong>选择检查项</strong><span>查看证据并形成现场结论</span></aside></ng-template>
    </section>
  `,
})
export class ChecksPage implements OnInit {
  readonly store = inject(CheckStore);
  readonly clearances = inject(ClearanceStore);
  private readonly fb = inject(FormBuilder);
  private readonly http = useHttp();
  private readonly dialog = inject(MatDialog);
  private readonly snack = inject(MatSnackBar);
  private readonly auth = useAuth();
  readonly pagination = usePagination(20);
  readonly canReview = this.auth.hasRole(ROLE.ADMIN, ROLE.SAFETY_MANAGER, ROLE.INSPECTOR);
  selected: SafetyCheck | null = null;
  filter = '';
  saving = false;
  readonly reviewForm = this.fb.nonNullable.group({
    result: ['passed' as 'passed' | 'failed', Validators.required],
    evidence: ['', Validators.required], remark: [''],
  });

  ngOnInit(): void { this.reload(); this.clearances.load(1, 200); }
  get pendingCount(): number { return this.store.summary().results.pending; }
  setFilter(result: string): void { this.filter = result; this.selected = null; this.pagination.reset(); this.reload(); }
  reload(): void { this.store.load(this.pagination.page(), this.pagination.pageSize(), undefined, this.filter); }
  pageChanged(event: PageEvent): void { this.selected = null; this.pagination.setPage(event.pageIndex + 1); this.pagination.pageSize.set(event.pageSize); this.reload(); }
  select(item: SafetyCheck): void { this.selected = item; this.reviewForm.reset({ result: 'passed', evidence: '', remark: '' }); }
  decisionFor(turnaroundId: number) { return this.clearances.items().find(item => item.turnaround_id === turnaroundId) ?? null; }

  review(): void {
    if (!this.selected || this.reviewForm.invalid) return;
    const value = this.reviewForm.getRawValue();
    const isFailed = value.result === 'failed';
    this.dialog.open(ConfirmDialogComponent, { data: {
      title: isFailed ? '确认检查不通过' : '确认检查通过',
      message: `结论提交后不可重复修改，证据将写入审计记录。`, confirmText: '提交结论', danger: isFailed,
    }}).afterClosed().subscribe(confirmed => {
      if (!confirmed || !this.selected) return;
      this.saving = true;
      checkReviewApi(this.http, this.selected.id, value.result, value.evidence.split(',').map(item => item.trim()).filter(Boolean), value.remark).subscribe({
        next: updated => { this.saving = false; this.selected = updated; this.reload(); this.clearances.load(1, 200); this.snack.open('检查结论已记录', '关闭', { duration: 2200 }); },
        error: error => { this.saving = false; this.snack.open(parseHttpError(error), '关闭', { duration: 4000 }); },
      });
    });
  }
}
