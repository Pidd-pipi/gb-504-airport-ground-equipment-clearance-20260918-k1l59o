import { CommonModule } from '@angular/common';
import { Component, Inject } from '@angular/core';
import { FormControl, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatRadioModule } from '@angular/material/radio';
import { BatchReviewPayload } from '../../api/check.api';
import { SafetyCheck } from '../../types';
import { RiskBadgeComponent } from './risk-badge.component';

@Component({
  selector: 'app-batch-review-dialog',
  standalone: true,
  imports: [
    CommonModule, ReactiveFormsModule, MatButtonModule, MatDialogModule, MatFormFieldModule, MatIconModule,
    MatInputModule, MatRadioModule, RiskBadgeComponent,
  ],
  template: `
    <h2 mat-dialog-title>
      <mat-icon>playlist_add_check</mat-icon>
      批量复核 {{ data.checks.length }} 项
    </h2>
    <mat-dialog-content>
      <p class="batch-hint">逐项选择复核结论，说明与证据对本批所有检查项共用；提交在同一事务完成，任一项已被复核或证据无效都会导致整批失败。</p>
      <div class="batch-items">
        <div class="batch-item" *ngFor="let item of data.checks">
          <div class="batch-item-copy">
            <strong>{{ item.item_name }}</strong>
            <small>{{ item.check_code }} · 周转 #{{ item.turnaround_id }}<span *ngIf="item.ground_unit_id"> · 设备 #{{ item.ground_unit_id }}</span></small>
          </div>
          <app-risk-badge [level]="item.risk_level"></app-risk-badge>
          <mat-radio-group [formControlName]="item.id">
            <mat-radio-button value="passed">通过</mat-radio-button>
            <mat-radio-button value="failed">未通过</mat-radio-button>
          </mat-radio-group>
        </div>
      </div>
      <mat-form-field appearance="outline">
        <mat-label>证据文件名 / 编号（多个用逗号分隔）</mat-label>
        <input matInput formControlName="evidence" placeholder="例如 GPU-test-0822.jpg, seal-0822.json">
        <mat-error>至少需要一个证据文件名 / 编号</mat-error>
      </mat-form-field>
      <mat-form-field appearance="outline">
        <mat-label>复核说明（共用）</mat-label>
        <textarea matInput rows="3" formControlName="remark" maxlength="1000"></textarea>
      </mat-form-field>
    </mat-dialog-content>
    <mat-dialog-actions align="end">
      <button mat-button (click)="dialog.close()">取消</button>
      <button mat-flat-button class="batch-submit" [disabled]="form.invalid" (click)="submit()">
        <mat-icon>task_alt</mat-icon>提交批量复核
      </button>
    </mat-dialog-actions>
  `,
  styles: [`
    h2 { display: flex; align-items: center; gap: 8px; font-size: 18px; }
    h2 mat-icon { color: #0d8b82; }
    mat-dialog-content { min-width: min(560px, 88vw); }
    .batch-hint { margin: 0 0 14px; padding: 10px 12px; border-left: 3px solid #0d9187; background: #f3f9f8; color: #5a6b71; font-size: 11px; line-height: 1.6; border-radius: 4px; }
    .batch-items { max-height: 280px; overflow: auto; margin-bottom: 14px; border: 1px solid #e1e7e8; border-radius: 5px; }
    .batch-item { display: grid; grid-template-columns: minmax(0, 1fr) auto minmax(190px, auto); align-items: center; gap: 12px; padding: 10px 14px; border-bottom: 1px solid #edf1f2; }
    .batch-item:last-child { border-bottom: 0; }
    .batch-item-copy strong { display: block; color: #2b3f46; font-size: 13px; }
    .batch-item-copy small { display: block; margin-top: 4px; color: #7a898e; font-size: 10px; }
    .batch-item mat-radio-group { display: flex; gap: 10px; }
    mat-form-field { width: 100%; }
    .batch-submit { background: #0c8e85; color: #fff; border-radius: 5px; }
  `],
})
export class BatchReviewDialogComponent {
  readonly form: FormGroup<Record<string, FormControl<string>>>;

  constructor(
    public readonly dialog: MatDialogRef<BatchReviewDialogComponent, BatchReviewPayload | undefined>,
    @Inject(MAT_DIALOG_DATA) public readonly data: { checks: SafetyCheck[] },
  ) {
    const controls: Record<string, FormControl<string>> = {
      evidence: new FormControl('', { nonNullable: true, validators: [Validators.required] }),
      remark: new FormControl('', { nonNullable: true }),
    };
    for (const check of data.checks) {
      controls[String(check.id)] = new FormControl<'passed' | 'failed'>('passed', { nonNullable: true, validators: [Validators.required] });
    }
    this.form = new FormGroup(controls);
  }

  submit(): void {
    if (this.form.invalid) return;
    const value = this.form.getRawValue();
    const evidence = value['evidence'].split(',').map(item => item.trim()).filter(Boolean);
    if (!evidence.length) return;
    const payload: BatchReviewPayload = {
      items: this.data.checks.map(check => ({ check_id: check.id, result: value[String(check.id)] as 'passed' | 'failed' })),
      evidence,
      remark: value['remark'],
    };
    this.dialog.close(payload);
  }
}
