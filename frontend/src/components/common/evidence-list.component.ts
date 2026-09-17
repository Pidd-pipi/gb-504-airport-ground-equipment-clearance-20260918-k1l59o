import { CommonModule } from '@angular/common';
import { Component, Input } from '@angular/core';
import { MatIconModule } from '@angular/material/icon';

@Component({
  selector: 'app-evidence-list',
  standalone: true,
  imports: [CommonModule, MatIconModule],
  template: `
    <div class="evidence" *ngIf="items.length; else empty">
      <span *ngFor="let item of items"><mat-icon>attach_file</mat-icon>{{ item }}</span>
    </div>
    <ng-template #empty><span class="empty">未上传证据</span></ng-template>
  `,
  styles: [`
    .evidence { display: flex; flex-wrap: wrap; gap: 6px; }
    .evidence span { display: inline-flex; align-items: center; gap: 2px; max-width: 220px; padding: 4px 7px; border: 1px solid #d8e0e2; border-radius: 4px; background: #fff; color: #486068; font-size: 11px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    mat-icon { font-size: 14px; width: 14px; height: 14px; }
    .empty { color: #8b989d; font-size: 12px; }
  `],
})
export class EvidenceListComponent {
  @Input() items: string[] = [];
}
