import { Component, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { MatTooltipModule } from '@angular/material/tooltip';

@Component({
  selector: 'app-page-header',
  standalone: true,
  imports: [CommonModule, MatIconModule, MatButtonModule, MatTooltipModule],
  template: `
    <div class="flex flex-col sm:flex-row sm:items-end justify-between gap-4 px-2 mb-4">
      <div class="space-y-1">
        <h1 class="text-3xl font-bold tracking-tight text-on-surface">{{ title() }}</h1>
        @if (subtitle()) {
          <p class="text-outline text-sm">{{ subtitle() }}</p>
        }
      </div>

      <div
        class="flex items-center gap-4 px-4 py-2 bg-surface-container-low rounded-2xl border border-outline-variant/30 text-xs text-outline font-medium"
      >
        <span class="flex items-center gap-1.5">
          <mat-icon class="!w-4 !h-4 !text-[14px]">{{ icon() }}</mat-icon>
          共 {{ total() }} {{ unit() }}
        </span>
        <button
          mat-icon-button
          icon-button-center
          (click)="refresh.emit()"
          [disabled]="loading()"
          matTooltip="刷新列表"
          class="!w-8 !h-8"
        >
          <mat-icon class="!text-[20px]">refresh</mat-icon>
        </button>
        <ng-content select="[chips]"></ng-content>
      </div>
    </div>
  `,
})
export class PageHeaderComponent {
  title = input.required<string>();
  subtitle = input<string>('');
  icon = input<string>('analytics');
  total = input<number>(0);
  unit = input<string>('条记录');
  loading = input<boolean>(false);
  refresh = output<void>();
}
