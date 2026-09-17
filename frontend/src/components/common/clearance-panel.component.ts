import { CommonModule } from '@angular/common';
import { Component, Input } from '@angular/core';
import { MatIconModule } from '@angular/material/icon';
import { ClearanceDecision } from '../../types';
import { StatusBadgeComponent } from './status-badge.component';
import { EvidenceListComponent } from './evidence-list.component';

@Component({
  selector: 'app-clearance-panel',
  standalone: true,
  imports: [CommonModule, MatIconModule, StatusBadgeComponent, EvidenceListComponent],
  template: `
    <section class="panel" [class.compact]="compact">
      <header><span><mat-icon>verified_user</mat-icon>放行状态</span><app-status-badge [value]="decision?.state || 'pending'"></app-status-badge></header>
      <div *ngIf="decision; else awaiting" class="body">
        <p><strong>决定依据</strong>{{ decision.reason || '等待放行员填写决定依据' }}</p>
        <p *ngIf="decision.restrictions"><strong>运行限制</strong>{{ decision.restrictions }}</p>
        <app-evidence-list [items]="decision.evidence || []"></app-evidence-list>
      </div>
      <ng-template #awaiting><p class="awaiting">检查完成后由安全放行员形成决定。</p></ng-template>
    </section>
  `,
  styles: [`
    .panel { border: 1px solid #d7e1e3; border-left: 3px solid #0e9187; background: #f8fbfb; border-radius: 5px; }
    header { min-height: 48px; padding: 0 14px; display: flex; align-items: center; justify-content: space-between; border-bottom: 1px solid #e0e7e8; }
    header > span { display: flex; align-items: center; gap: 7px; font-weight: 600; color: #273940; }
    header mat-icon { color: #0d8b82; font-size: 19px; width: 19px; height: 19px; }
    .body { padding: 12px 14px; } p { margin: 0 0 10px; color: #576a71; font-size: 12px; line-height: 1.5; }
    p strong { display: block; color: #263a41; margin-bottom: 2px; }
    .awaiting { padding: 13px 14px; margin: 0; }
    .compact header { min-height: 40px; } .compact .body { padding: 9px 12px; }
  `],
})
export class ClearancePanelComponent {
  @Input() decision: ClearanceDecision | null = null;
  @Input() compact = false;
}
