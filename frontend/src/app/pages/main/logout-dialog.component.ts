import { Component } from '@angular/core';
import { MatDialogModule } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';

@Component({
  selector: 'app-logout-dialog',
  standalone: true,
  imports: [MatDialogModule, MatButtonModule, MatIconModule],
  template: `
    <div class="p-2">
      <h2 mat-dialog-title class="!flex !items-center !gap-2">
        <mat-icon color="warn">logout</mat-icon>
        确认注销
      </h2>
      <mat-dialog-content>
        <p class="py-2 text-slate-600">确定要注销并退出系统吗？所有的未保存更改可能会丢失。</p>
      </mat-dialog-content>
      <mat-dialog-actions align="end" class="!pb-2">
        <button mat-button mat-dialog-close>取消</button>
        <button mat-flat-button color="warn" [mat-dialog-close]="true" class="!px-6">确认注销</button>
      </mat-dialog-actions>
    </div>
  `,
})
export class LogoutDialogComponent {}
