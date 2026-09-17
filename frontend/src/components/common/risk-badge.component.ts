import { Component, Input } from '@angular/core';
import { MatIconModule } from '@angular/material/icon';
import { RiskLevel } from '../../types';
import { RISK_TEXT } from '../../constants/enums';

@Component({
  selector: 'app-risk-badge',
  standalone: true,
  template: `<span class="risk" [attr.data-risk]="level"><mat-icon>{{ level === 'critical' || level === 'high' ? 'warning' : 'shield' }}</mat-icon>{{ text[level] }}风险</span>`,
  imports: [MatIconModule],
  styles: [`
    .risk { display: inline-flex; align-items: center; gap: 4px; height: 24px; padding: 0 8px; border-radius: 4px; color: #446068; background: #edf2f3; font-size: 12px; white-space: nowrap; }
    mat-icon { font-family: 'Material Icons'; font-size: 14px; width: 14px; height: 14px; }
    [data-risk="high"] { color: #a04b00; background: #fff0de; }
    [data-risk="critical"] { color: #b42318; background: #fdecea; font-weight: 600; }
    [data-risk="low"] { color: #18794e; background: #e7f6ed; }
  `],
})
export class RiskBadgeComponent {
  @Input({ required: true }) level: RiskLevel = 'medium';
  readonly text = RISK_TEXT;
}
