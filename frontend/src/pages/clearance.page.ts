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
import { clearanceDecideApi } from '../api/clearance.api';
import { ClearancePanelComponent } from '../components/common/clearance-panel.component';
import { ConfirmDialogComponent } from '../components/common/confirm-dialog.component';
import { StatusBadgeComponent } from '../components/common/status-badge.component';
import { CLEARANCE_STATE_TEXT, ROLE } from '../constants/enums';
import { useAuth } from '../hooks/use-auth';
import { usePagination } from '../hooks/use-pagination';
import { ClearanceStore } from '../stores/clearance.store';
import { TurnaroundStore } from '../stores/turnaround.store';
import { ClearanceDecision, ClearanceState } from '../types';
import { parseHttpError, useHttp } from '../utils/request';

@Component({
  selector: 'app-clearance-page',
  standalone: true,
  imports: [
    CommonModule, ReactiveFormsModule, MatButtonModule, MatDialogModule, MatFormFieldModule, MatIconModule,
    MatInputModule, MatPaginatorModule, MatProgressBarModule, MatSelectModule, MatSnackBarModule, ClearancePanelComponent, StatusBadgeComponent,
  ],
  template: `
    <header class="page-head">
      <div><p>CLEARANCE REVIEW</p><h1>安全放行审核</h1><span>检查证据完整后形成放行、限制或撤销决定</span></div>
      <div class="head-status"><mat-icon>policy</mat-icon>状态迁移全审计</div>
    </header>
    <section class="metrics">
      <div><span>待决定</span><strong>{{ store.summary().states.pending }}</strong><small>等待检查闭环</small></div>
      <div><span>已放行</span><strong class="good">{{ store.summary().states.cleared }}</strong><small>无阻断条件</small></div>
      <div><span>限制放行</span><strong>{{ store.summary().states.restricted }}</strong><small>附带运行条件</small></div>
      <div><span>已撤销</span><strong class="danger">{{ store.summary().states.revoked }}</strong><small>禁止继续周转</small></div>
    </section>

    <section class="clearance-layout">
      <div class="decision-list">
        <div class="table-tools">
          <mat-form-field appearance="outline" subscriptSizing="dynamic"><mat-label>放行状态</mat-label><mat-select [(value)]="filter" (selectionChange)="resetAndLoad()"><mat-option value="">全部状态</mat-option><mat-option value="pending">待决定</mat-option><mat-option value="cleared">已放行</mat-option><mat-option value="restricted">限制放行</mat-option><mat-option value="revoked">已撤销</mat-option></mat-select></mat-form-field>
        </div>
        <mat-progress-bar *ngIf="store.loading()" mode="indeterminate"></mat-progress-bar>
        <button class="decision-row" *ngFor="let item of store.items()" [class.selected]="selected?.id === item.id" (click)="select(item)">
          <span class="flight-mark"><mat-icon>flight</mat-icon></span>
          <span><strong>{{ flightLabel(item.turnaround_id) }}</strong><small>周转 #{{ item.turnaround_id }} · 决定 #{{ item.id }}</small></span>
          <app-status-badge [value]="item.state"></app-status-badge><mat-icon>chevron_right</mat-icon>
        </button>
        <div class="empty" *ngIf="!store.loading() && !store.items().length"><mat-icon>verified_user</mat-icon><strong>暂无放行记录</strong><span>建立周转后会自动生成待决定项</span></div>
        <mat-paginator [length]="store.total()" [pageIndex]="pagination.page() - 1" [pageSize]="pagination.pageSize()" [pageSizeOptions]="[10, 20, 50, 100]" (page)="pageChanged($event)"></mat-paginator>
      </div>

      <aside class="decision-pane" *ngIf="selected; else selectionHint">
        <app-clearance-panel [decision]="selected"></app-clearance-panel>
        <div class="decision-context">
          <span><small>航班周转</small>{{ flightLabel(selected.turnaround_id) }}</span>
          <span><small>前一状态</small>{{ selected.previous_state ? stateText[selected.previous_state] : '首次决定' }}</span>
          <span><small>操作人员</small>{{ selected.operator_id ? '#' + selected.operator_id : '待分配' }}</span>
        </div>
        <form *ngIf="canManage && selected.state !== 'revoked'" [formGroup]="form" (ngSubmit)="decide()" class="decision-form">
          <h3>{{ selected.state === 'pending' ? '形成放行决定' : '变更为撤销状态' }}</h3>
          <mat-form-field appearance="outline"><mat-label>目标状态</mat-label><mat-select formControlName="state"><mat-option *ngFor="let state of targetStates" [value]="state">{{ stateText[state] }}</mat-option></mat-select></mat-form-field>
          <mat-form-field appearance="outline" *ngIf="form.controls.state.value === 'restricted'"><mat-label>运行限制</mat-label><textarea matInput rows="2" formControlName="restrictions" placeholder="例如：仅允许低速牵引，不得接入地面电源"></textarea></mat-form-field>
          <mat-form-field appearance="outline"><mat-label>决定依据</mat-label><textarea matInput rows="3" formControlName="reason"></textarea></mat-form-field>
          <mat-form-field appearance="outline"><mat-label>证据编号 / 文件名</mat-label><input matInput formControlName="evidence" placeholder="多个证据用逗号分隔"></mat-form-field>
          <div class="form-warning" *ngIf="form.controls.state.value === 'revoked'"><mat-icon>warning</mat-icon>撤销后不可恢复，请确认已通知现场调度。</div>
          <button mat-flat-button type="submit" [class.danger-button]="form.controls.state.value === 'revoked'" [disabled]="form.invalid || saving"><mat-icon>gavel</mat-icon>记录决定</button>
        </form>
      </aside>
      <ng-template #selectionHint><aside class="decision-pane placeholder"><mat-icon>verified_user</mat-icon><strong>选择放行记录</strong><span>查看当前状态并形成安全决定</span></aside></ng-template>
    </section>
  `,
})
export class ClearancePage implements OnInit {
  readonly store = inject(ClearanceStore);
  readonly turnarounds = inject(TurnaroundStore);
  private readonly fb = inject(FormBuilder);
  private readonly http = useHttp();
  private readonly dialog = inject(MatDialog);
  private readonly snack = inject(MatSnackBar);
  private readonly auth = useAuth();
  readonly pagination = usePagination(20);
  readonly canManage = this.auth.hasRole(ROLE.ADMIN, ROLE.SAFETY_MANAGER);
  readonly stateText = CLEARANCE_STATE_TEXT;
  selected: ClearanceDecision | null = null;
  filter = '';
  saving = false;
  targetStates: Array<Exclude<ClearanceState, 'pending'>> = ['cleared', 'restricted', 'revoked'];
  readonly form = this.fb.nonNullable.group({
    state: ['cleared' as Exclude<ClearanceState, 'pending'>, Validators.required],
    restrictions: [''], reason: ['', Validators.required], evidence: ['', Validators.required],
  });

  ngOnInit(): void { this.reload(); this.turnarounds.load(1, 200); }
  flightLabel(turnaroundId: number): string {
    const row = this.turnarounds.items().find(item => item.id === turnaroundId);
    return row ? `${row.flight_no} / ${row.stand}` : `周转 #${turnaroundId}`;
  }
  reload(): void { this.selected = null; this.store.load(this.pagination.page(), this.pagination.pageSize(), this.filter); }
  resetAndLoad(): void { this.pagination.reset(); this.reload(); }
  pageChanged(event: PageEvent): void { this.selected = null; this.pagination.setPage(event.pageIndex + 1); this.pagination.pageSize.set(event.pageSize); this.reload(); }
  select(item: ClearanceDecision): void {
    this.selected = item;
    this.targetStates = item.state === 'pending' ? ['cleared', 'restricted', 'revoked'] : ['revoked'];
    this.form.reset({ state: this.targetStates[0], restrictions: '', reason: '', evidence: '' });
  }

  decide(): void {
    if (!this.selected || this.form.invalid) return;
    const value = this.form.getRawValue();
    if (value.state === 'restricted' && !value.restrictions.trim()) {
      this.snack.open('限制放行必须填写运行限制', '关闭', { duration: 3200 });
      return;
    }
    const danger = value.state === 'revoked';
    this.dialog.open(ConfirmDialogComponent, { data: {
      title: `确认${this.stateText[value.state]}`, message: `该决定将立即写入状态迁移和审计证据，确认继续？`,
      confirmText: '确认记录', danger,
    }}).afterClosed().subscribe(confirmed => {
      if (!confirmed || !this.selected) return;
      this.saving = true;
      clearanceDecideApi(this.http, {
        turnaround_id: this.selected.turnaround_id, state: value.state, restrictions: value.restrictions,
        reason: value.reason, evidence: value.evidence.split(',').map(item => item.trim()).filter(Boolean),
      }).subscribe({
        next: updated => { this.saving = false; this.selected = updated; this.store.load(this.pagination.page(), this.pagination.pageSize(), this.filter); this.snack.open('安全放行决定已记录', '关闭', { duration: 2500 }); },
        error: error => { this.saving = false; this.snack.open(parseHttpError(error), '关闭', { duration: 4500 }); },
      });
    });
  }
}
