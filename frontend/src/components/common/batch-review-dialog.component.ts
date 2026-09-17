import { CommonModule } from '@angular/common';
import { Component, Inject } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatRadioModule } from '@angular/material/radio';
import { SafetyCheck } from '../../types';

export interface BatchReviewDialogResult {
  items: { id: number; result: 'passed' | 'failed' }[];
  evidence: string[];
  remark: string;
}

@Component({
  selector: 'app-batch-review-dialog',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule, MatButtonModule, MatDialogModule, MatFormFieldModule, MatIconModule, MatInputModule, MatRadioModule],
  template: `
    <h2 mat-dialog-title><mat-icon>rule_folder</mat-icon>批量复核 {{ data.checks.length }} 项</h2>
    <mat-dialog-content>
      <p class="hint">逐项给出结论，所有项目共用同一份证据与说明；任一项已被他人复核时整批提交失败，不会产生部分结论。</p>
      <div class="decision-list">
        <div class="decision-row" *ngFor="let check of data.checks">
          <div class="decision-copy">
            <strong>{{ check.item_name }}</strong>
            <small>{{ check.check_code }} · 周转 #{{ check.turnaround_id }} · 序号 {{ check.sequence | number:'2.0' }}</small>
          </div>
          <mat-radio-group [value]="decisions[check.id] || ''" (change)="setDecision(check.id, $event.value)" class="decision-toggle">
            <mat-radio-button value="passed">通过</mat-radio-button>
            <mat-radio-button value="failed" class="fail">未通过</mat-radio-button>
          </mat-radio-group>
        </div>
      </div>
      <form [formGroup]="form" (ngSubmit)="submit()" class="shared-form">
        <mat-form-field appearance="outline">
          <mat-label>共用证据文件名 / 编号（逗号分隔）</mat-label>
          <input matInput formControlName="evidence" placeholder="例如 GPU-test-0822.jpg, seal-check-03.jpg">
        </mat-form-field>
        <mat-form-field appearance="outline">
          <mat-label>共用检查说明</mat-label>
          <textarea matInput rows="3" formControlName="remark" maxlength="1000"></textarea>
        </mat-form-field>
        <p class="form-error" *ngIf="submitted && form.controls.evidence.invalid">至少填写一条证据引用</p>
        <p class="form-error" *ngIf="submitted && pendingDecisions > 0">还有 {{ pendingDecisions }} 项未选择通过 / 未通过</p>
      </form>
    </mat-dialog-content>
    <mat-dialog-actions align="end">
      <button mat-button (click)="dialog.close(null)">取消</button>
      <button mat-flat-button type="button" (click)="submit()"><mat-icon>task_alt</mat-icon>整批提交</button>
    </mat-dialog-actions>
  `,
  styles: [`
    h2 { display: flex; align-items: center; gap: 8px; font-size: 18px; }
    h2 mat-icon { color: #0d8b82; }
    mat-dialog-content { min-width: min(560px, 82vw); max-height: 68vh; }
    .hint { margin: 0 0 12px; color: #5d6e74; font-size: 12px; }
    .decision-list { border: 1px solid #e1e7e8; border-radius: 6px; overflow: hidden; margin-bottom: 14px; }
    .decision-row { display: flex; justify-content: space-between; align-items: center; gap: 12px; padding: 10px 14px; border-bottom: 1px solid #eef2f3; }
    .decision-row:last-child { border-bottom: 0; }
    .decision-copy strong, .decision-copy small { display: block; }
    .decision-copy strong { font-size: 13px; color: #273b42; }
    .decision-copy small { color: #76868c; font-size: 11px; margin-top: 2px; }
    .decision-toggle { display: flex; gap: 6px; flex-shrink: 0; }
    .shared-form mat-form-field { width: 100%; }
    .form-error { margin: -6px 0 8px; color: #b42318; font-size: 12px; }
    button[mat-flat-button] { background: #0c8e85; color: #fff; border-radius: 5px; }
  `],
})
export class BatchReviewDialogComponent {
  readonly form = this.fb.nonNullable.group({
    evidence: ['', Validators.required],
    remark: [''],
  });
  readonly decisions: Record<number, 'passed' | 'failed'> = {};
  submitted = false;

  constructor(
    private readonly fb: FormBuilder,
    public readonly dialog: MatDialogRef<BatchReviewDialogComponent>,
    @Inject(MAT_DIALOG_DATA) public readonly data: { checks: SafetyCheck[] },
  ) {}

  setDecision(id: number, result: 'passed' | 'failed'): void {
    this.decisions[id] = result;
  }

  get pendingDecisions(): number {
    return this.data.checks.filter(check => !this.decisions[check.id]).length;
  }

  submit(): void {
    this.submitted = true;
    if (this.form.controls.evidence.invalid || this.pendingDecisions > 0) {
      return;
    }
    const value = this.form.getRawValue();
    const evidence = value.evidence.split(',').map(item => item.trim()).filter(Boolean);
    const items = this.data.checks.map(check => ({ id: check.id, result: this.decisions[check.id] }));
    this.dialog.close({ items, evidence, remark: value.remark } satisfies BatchReviewDialogResult);
  }
}
