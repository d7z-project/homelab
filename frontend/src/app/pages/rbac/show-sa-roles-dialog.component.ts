import { Component, Inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MAT_DIALOG_DATA, MatDialogModule } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatListModule } from '@angular/material/list';
import { MatDividerModule } from '@angular/material/divider';
import { ModelsRole } from '../../generated';

@Component({
  selector: 'app-show-sa-roles-dialog',
  standalone: true,
  imports: [
    CommonModule,
    MatDialogModule,
    MatButtonModule,
    MatIconModule,
    MatListModule,
    MatDividerModule,
  ],
  template: `
    <h2 mat-dialog-title class="!pt-6">生效权限详情</h2>
    <mat-dialog-content>
      <div class="py-2">
        <p class="text-sm text-outline mb-4">
          账号 ID: <strong class="font-mono">{{ data.saID }}</strong> 当前拥有的权限详情：
        </p>

        @if (data.roles.length > 0) {
          <div class="space-y-4">
            @for (role of data.roles; track role.id) {
              <div
                class="border border-outline-variant rounded-2xl overflow-hidden bg-surface-container-lowest"
              >
                <div class="bg-surface-container px-4 py-2 flex items-center gap-2">
                  <mat-icon
                    class="!text-secondary !w-[18px] !h-[18px] !text-[18px] !flex !items-center !justify-center"
                    >shield</mat-icon
                  >
                  <span class="font-bold text-sm text-on-surface">{{ role.name }}</span>
                  <span class="text-[10px] text-outline font-mono ml-auto opacity-60">{{ role.id }}</span>
                </div>
                <div class="p-4 space-y-3">
                  @for (rule of role.rules; track rule) {
                    <div class="flex flex-wrap items-center gap-2">
                      <div class="flex gap-1">
                        <span
                          class="text-primary text-[10px] font-bold uppercase border border-primary/20 px-1.5 rounded bg-primary/5"
                          >{{ rule.resource }}</span
                        >
                      </div>
                      <mat-icon
                        class="!w-[12px] !h-[12px] !text-[12px] text-outline opacity-50 !flex !items-center !justify-center"
                        >east</mat-icon
                      >
                      <div class="flex gap-1">
                        @for (verb of rule.verbs; track verb) {
                          <span
                            class="text-secondary text-[10px] font-bold uppercase border border-secondary/20 px-1.5 rounded bg-secondary/5"
                            >{{ verb }}</span
                          >
                        }
                      </div>
                    </div>
                  }
                </div>
              </div>
            }
          </div>
        } @else {
          <div
            class="flex flex-col items-center justify-center py-10 bg-surface-container rounded-xl border border-dashed border-outline-variant"
          >
            <mat-icon
              class="text-outline opacity-20 !w-[48px] !h-[48px] !text-[48px] !flex !items-center !justify-center mb-2"
              >block</mat-icon
            >
            <p class="text-xs text-outline italic">该账号目前没有任何生效的角色绑定</p>
          </div>
        }
      </div>
    </mat-dialog-content>
    <mat-dialog-actions align="end" class="!px-6 !pb-6">
      <button mat-button mat-dialog-close color="primary">关闭</button>
    </mat-dialog-actions>
  `,
})
export class ShowSaRolesDialogComponent {
  constructor(@Inject(MAT_DIALOG_DATA) public data: { saID: string; roles: ModelsRole[] }) {}
}
