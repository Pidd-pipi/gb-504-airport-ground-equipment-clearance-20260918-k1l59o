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
import { MatTableModule } from '@angular/material/table';
import { groundUnitCreateApi, groundUnitStateApi } from '../api/ground-unit.api';
import { ConfirmDialogComponent } from '../components/common/confirm-dialog.component';
import { StatusBadgeComponent } from '../components/common/status-badge.component';
import { ROLE, UNIT_STATE_TEXT } from '../constants/enums';
import { useAuth } from '../hooks/use-auth';
import { usePagination } from '../hooks/use-pagination';
import { GroundUnitStore } from '../stores/ground-unit.store';
import { GroundUnit, UnitState } from '../types';
import { parseHttpError, useHttp } from '../utils/request';

@Component({
  selector: 'app-ground-units-page',
  standalone: true,
  imports: [
    CommonModule, ReactiveFormsModule, MatButtonModule, MatDialogModule, MatFormFieldModule, MatIconModule,
    MatInputModule, MatPaginatorModule, MatProgressBarModule, MatSelectModule, MatSnackBarModule, MatTableModule, StatusBadgeComponent,
  ],
  template: `
    <header class="page-head">
      <div><p>GROUND UNIT CONTROL</p><h1>地面设备状态</h1><span>设备可用性直接参与周转编排与放行判断</span></div>
      <button *ngIf="canManage" mat-flat-button (click)="showCreate = !showCreate"><mat-icon>{{ showCreate ? 'close' : 'add' }}</mat-icon>{{ showCreate ? '收起' : '登记设备' }}</button>
    </header>
    <section class="metrics">
      <div><span>设备总数</span><strong>{{ store.summary().total }}</strong><small>在册地面单元</small></div>
      <div><span>可投入</span><strong class="good">{{ store.summary().states.available }}</strong><small>当前班次可用</small></div>
      <div><span>检查中</span><strong>{{ store.summary().states.inspection }}</strong><small>暂不参与编排</small></div>
      <div><span>已锁定</span><strong class="danger">{{ store.summary().states.blocked }}</strong><small>存在阻断风险</small></div>
    </section>

    <section *ngIf="showCreate" class="create-band">
      <div class="band-title"><mat-icon>airport_shuttle</mat-icon><span><strong>登记地面设备</strong><small>设备编号需保持唯一</small></span></div>
      <form [formGroup]="form" (ngSubmit)="create()">
        <mat-form-field appearance="outline"><mat-label>设备编号</mat-label><input matInput formControlName="unit_code"></mat-form-field>
        <mat-form-field appearance="outline"><mat-label>设备名称</mat-label><input matInput formControlName="name"></mat-form-field>
        <mat-form-field appearance="outline"><mat-label>设备类型</mat-label><mat-select formControlName="unit_type"><mat-option value="tug">牵引车</mat-option><mat-option value="gpu">地面电源</mat-option><mat-option value="belt_loader">传送带</mat-option><mat-option value="water_service">清水车</mat-option><mat-option value="catering">配餐车</mat-option></mat-select></mat-form-field>
        <mat-form-field appearance="outline"><mat-label>所在机位</mat-label><input matInput formControlName="stand"></mat-form-field>
        <mat-form-field appearance="outline"><mat-label>初始状态</mat-label><mat-select formControlName="state"><mat-option *ngFor="let state of states" [value]="state">{{ stateText[state] }}</mat-option></mat-select></mat-form-field>
        <mat-form-field appearance="outline" class="wide"><mat-label>状态备注</mat-label><input matInput formControlName="notes"></mat-form-field>
        <div class="form-actions"><button mat-button type="button" (click)="showCreate = false">取消</button><button mat-flat-button type="submit" [disabled]="form.invalid || saving">保存设备</button></div>
      </form>
    </section>

    <section class="table-section">
      <div class="table-tools">
        <mat-form-field appearance="outline" subscriptSizing="dynamic"><mat-label>状态</mat-label><mat-select [(value)]="stateFilter" (selectionChange)="searchText = search.value; resetAndLoad()"><mat-option value="">全部状态</mat-option><mat-option *ngFor="let state of states" [value]="state">{{ stateText[state] }}</mat-option></mat-select></mat-form-field>
        <mat-form-field appearance="outline" subscriptSizing="dynamic"><mat-icon matPrefix>search</mat-icon><input #search matInput placeholder="编号、名称或机位" (keyup.enter)="searchText = search.value; resetAndLoad()"></mat-form-field>
      </div>
      <mat-progress-bar *ngIf="store.loading()" mode="indeterminate"></mat-progress-bar>
      <p class="error" *ngIf="store.error()">{{ store.error() }}</p>
      <div class="table-wrap">
        <table mat-table [dataSource]="store.items()">
          <ng-container matColumnDef="unit"><th mat-header-cell *matHeaderCellDef>设备</th><td mat-cell *matCellDef="let row"><strong>{{ row.unit_code }}</strong><small>{{ row.name }}</small></td></ng-container>
          <ng-container matColumnDef="type"><th mat-header-cell *matHeaderCellDef>类型</th><td mat-cell *matCellDef="let row">{{ typeText(row.unit_type) }}</td></ng-container>
          <ng-container matColumnDef="stand"><th mat-header-cell *matHeaderCellDef>机位</th><td mat-cell *matCellDef="let row">{{ row.stand }}</td></ng-container>
          <ng-container matColumnDef="inspection"><th mat-header-cell *matHeaderCellDef>最近检查</th><td mat-cell *matCellDef="let row">{{ row.last_inspection_at ? (row.last_inspection_at | date:'MM-dd HH:mm') : '未记录' }}</td></ng-container>
          <ng-container matColumnDef="state"><th mat-header-cell *matHeaderCellDef>状态</th><td mat-cell *matCellDef="let row"><app-status-badge [value]="row.state"></app-status-badge><small class="note">{{ row.notes }}</small></td></ng-container>
          <ng-container matColumnDef="actions"><th mat-header-cell *matHeaderCellDef></th><td mat-cell *matCellDef="let row"><div class="row-actions"><button *ngIf="canReport && row.state !== 'blocked' && row.state !== 'retired'" mat-stroked-button color="warn" (click)="changeState(row, 'blocked')">锁定</button><button *ngIf="canManage && (row.state === 'blocked' || row.state === 'inspection')" mat-stroked-button (click)="changeState(row, 'available')">恢复可用</button></div></td></ng-container>
          <tr mat-header-row *matHeaderRowDef="columns"></tr><tr mat-row *matRowDef="let row; columns: columns"></tr>
        </table>
        <div class="empty" *ngIf="!store.loading() && !store.items().length"><mat-icon>airport_shuttle</mat-icon><strong>暂无设备记录</strong><span>调整筛选条件或登记设备</span></div>
      </div>
      <mat-paginator [length]="store.total()" [pageIndex]="pagination.page() - 1" [pageSize]="pagination.pageSize()" [pageSizeOptions]="[10, 20, 50, 100]" (page)="pageChanged($event)"></mat-paginator>
    </section>
  `,
})
export class GroundUnitsPage implements OnInit {
  readonly store = inject(GroundUnitStore);
  private readonly fb = inject(FormBuilder);
  private readonly http = useHttp();
  private readonly dialog = inject(MatDialog);
  private readonly snack = inject(MatSnackBar);
  private readonly auth = useAuth();
  readonly pagination = usePagination(20);
  readonly canManage = this.auth.hasRole(ROLE.ADMIN, ROLE.SAFETY_MANAGER);
  readonly canReport = this.auth.hasRole(ROLE.ADMIN, ROLE.SAFETY_MANAGER, ROLE.INSPECTOR);
  readonly columns = ['unit', 'type', 'stand', 'inspection', 'state', 'actions'];
  readonly states: UnitState[] = ['available', 'inspection', 'blocked', 'retired'];
  readonly stateText = UNIT_STATE_TEXT;
  stateFilter = '';
  searchText = '';
  showCreate = false;
  saving = false;
  readonly form = this.fb.nonNullable.group({
    unit_code: ['', Validators.required], name: ['', Validators.required], unit_type: ['tug', Validators.required],
    stand: ['', Validators.required], state: ['available' as UnitState, Validators.required], notes: [''],
  });

  ngOnInit(): void { this.reload(); }
  reload(): void { this.store.load(this.pagination.page(), this.pagination.pageSize(), this.stateFilter, this.searchText); }
  resetAndLoad(): void { this.pagination.reset(); this.reload(); }
  pageChanged(event: PageEvent): void { this.pagination.setPage(event.pageIndex + 1); this.pagination.pageSize.set(event.pageSize); this.reload(); }
  typeText(type: string): string { return ({ tug: '牵引车', gpu: '地面电源', belt_loader: '行李传送带', water_service: '清水车', catering: '配餐车' } as Record<string, string>)[type] || type; }

  create(): void {
    if (this.form.invalid) return;
    this.saving = true;
    const value = this.form.getRawValue();
    groundUnitCreateApi(this.http, { ...value, unit_code: value.unit_code.trim().toUpperCase(), stand: value.stand.trim().toUpperCase() }).subscribe({
      next: () => { this.saving = false; this.showCreate = false; this.form.reset({ unit_code: '', name: '', unit_type: 'tug', stand: '', state: 'available', notes: '' }); this.reload(); this.snack.open('设备已登记', '关闭', { duration: 2200 }); },
      error: error => { this.saving = false; this.snack.open(parseHttpError(error), '关闭', { duration: 4000 }); },
    });
  }

  changeState(unit: GroundUnit, state: UnitState): void {
    const label = state === 'blocked' ? '锁定' : '恢复';
    this.dialog.open(ConfirmDialogComponent, { data: { title: `${label}设备`, message: `确认将 ${unit.unit_code} 变更为“${this.stateText[state]}”？`, danger: state === 'blocked' } }).afterClosed().subscribe(confirmed => {
      if (!confirmed) return;
      groundUnitStateApi(this.http, unit.id, state, state === 'blocked' ? '现场检查发现异常，暂停投入' : '复检通过，恢复可用', unit.version).subscribe({
        next: () => { this.reload(); this.snack.open(`设备已${label}`, '关闭', { duration: 2200 }); },
        error: error => this.snack.open(parseHttpError(error), '关闭', { duration: 4000 }),
      });
    });
  }
}
