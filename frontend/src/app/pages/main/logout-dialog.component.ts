import { Component } from '@angular/core';
import { MatDialogModule } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';

@Component({
  selector: 'app-logout-dialog',
  standalone: true,
  imports: [MatDialogModule, MatButtonModule, MatIconModule],
  template: `
    <h2 mat-dialog-title class="!flex !items-center !gap-3 !pt-6">
      <mat-icon class="!text-error !w-6 !h-6 !text-[24px]">logout</mat-icon>
      确认注销
    </h2>
    <mat-dialog-content>
      <p class="py-3 text-on-surface opacity-80">确定要注销并退出系统吗？所有的未保存更改可能会丢失。</p>
    </mat-dialog-content>
    <mat-dialog-actions align="end" class="!px-6 !pb-6">
      <button mat-button mat-dialog-close>取消</button>
      <button mat-flat-button color="warn" [mat-dialog-close]="true" class="!ml-2">确认注销</button>
    </mat-dialog-actions>
  `,
})
export class LogoutDialogComponent {}
