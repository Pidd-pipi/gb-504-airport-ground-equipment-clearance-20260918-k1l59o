import { Component, Input } from '@angular/core';
import { STATUS_TEXT } from '../../constants/enums';

@Component({
  selector: 'app-status-badge',
  standalone: true,
  template: `<span class="status" [attr.data-state]="value"><i></i>{{ label || text[value] || value }}</span>`,
  styles: [`
    .status { display: inline-flex; align-items: center; gap: 6px; min-height: 24px; padding: 0 8px; border-radius: 4px; background: #eef2f3; color: #52636a; font-size: 12px; white-space: nowrap; }
    i { width: 6px; height: 6px; border-radius: 50%; background: currentColor; }
    [data-state="available"], [data-state="passed"], [data-state="cleared"], [data-state="completed"] { color: #18794e; background: #e7f6ed; }
    [data-state="inspection"], [data-state="checking"], [data-state="pending"], [data-state="open"] { color: #8a5b00; background: #fff4d6; }
    [data-state="blocked"], [data-state="failed"], [data-state="revoked"] { color: #b42318; background: #fdecea; }
    [data-state="restricted"], [data-state="decisioned"] { color: #1d5d91; background: #e8f2fb; }
    [data-state="retired"] { color: #66747a; background: #e8ebec; }
  `],
})
export class StatusBadgeComponent {
  @Input({ required: true }) value = '';
  @Input() label = '';
  readonly text = STATUS_TEXT;
}
