import { Component, Inject } from '@angular/core';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';

export interface ConfirmDialogData {
  title: string;
  message: string;
  confirmText?: string;
  danger?: boolean;
}

@Component({
  selector: 'app-confirm-dialog',
  standalone: true,
  imports: [MatDialogModule, MatButtonModule, MatIconModule],
  template: `
    <h2 mat-dialog-title><mat-icon [class.danger]="data.danger">{{ data.danger ? 'warning' : 'help' }}</mat-icon>{{ data.title }}</h2>
    <mat-dialog-content>{{ data.message }}</mat-dialog-content>
    <mat-dialog-actions align="end">
      <button mat-button (click)="dialog.close(false)">取消</button>
      <button mat-flat-button [class.danger-button]="data.danger" (click)="dialog.close(true)">{{ data.confirmText || '确认' }}</button>
    </mat-dialog-actions>
  `,
  styles: [`
    h2 { display: flex; align-items: center; gap: 8px; font-size: 18px; } h2 mat-icon { color: #0d8b82; } h2 mat-icon.danger { color: #b42318; }
    mat-dialog-content { min-width: min(360px, 70vw); color: #53656c; }
    button[mat-flat-button] { background: #0d8b82; color: #fff; } button.danger-button { background: #b42318; }
  `],
})
export class ConfirmDialogComponent {
  constructor(
    public readonly dialog: MatDialogRef<ConfirmDialogComponent>,
    @Inject(MAT_DIALOG_DATA) public readonly data: ConfirmDialogData,
  ) {}
}
