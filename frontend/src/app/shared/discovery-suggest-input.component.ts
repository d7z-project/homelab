import { Component, Input, OnInit, forwardRef, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  ControlValueAccessor,
  NG_VALUE_ACCESSOR,
  FormsModule,
  ReactiveFormsModule,
  FormControl,
} from '@angular/forms';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatAutocompleteModule } from '@angular/material/autocomplete';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { DiscoveryService, ModelsLookupItem } from '../generated';
import { debounceTime, distinctUntilChanged, switchMap, of, catchError, finalize, map } from 'rxjs';

@Component({
  selector: 'app-discovery-suggest-input',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ReactiveFormsModule,
    MatFormFieldModule,
    MatInputModule,
    MatAutocompleteModule,
    MatIconModule,
    MatProgressBarModule,
  ],
  providers: [
    {
      provide: NG_VALUE_ACCESSOR,
      useExisting: forwardRef(() => DiscoverySuggestInputComponent),
      multi: true,
    },
  ],
  template: `
    <mat-form-field
      [appearance]="appearance"
      class="w-full relative"
      [subscriptSizing]="subscriptSizing"
    >
      <mat-label>{{ label }}</mat-label>
      <input
        matInput
        [placeholder]="placeholder"
        [matAutocomplete]="auto"
        [formControl]="control"
      />

      <!-- Loading bar at bottom of field -->
      <div
        class="absolute left-0 right-0 bottom-0 overflow-hidden transition-all duration-300 pointer-events-none"
        [style.height]="isLoading() ? '2px' : '0px'"
        [style.opacity]="isLoading() ? 1 : 0"
      >
        <mat-progress-bar mode="indeterminate" class="!h-[2px]"></mat-progress-bar>
      </div>

      <mat-autocomplete #auto="matAutocomplete">
        <!-- Variable/Ref Suggestions (Priority) -->
        @if (staticSuggestions.length > 0) {
          <mat-optgroup label="变量引用">
            @for (ref of staticSuggestions; track ref) {
              <mat-option [value]="ref">
                <div class="flex items-center gap-2">
                  <mat-icon class="!text-[14px] !w-4 !h-4 opacity-50">data_object</mat-icon>
                  <span class="font-mono text-xs">{{ ref }}</span>
                </div>
              </mat-option>
            }
          </mat-optgroup>
        }

        <!-- Discovery Results -->
        @if (items().length > 0) {
          <mat-optgroup [label]="lookupLabel">
            @for (item of items(); track item.id) {
              <mat-option [value]="item.id">
                <div class="flex flex-col leading-tight py-1">
                  <div class="flex items-center justify-between gap-3">
                    <div class="flex items-center gap-2">
                      @if (item.icon) {
                        <mat-icon class="!text-sm !w-4 !h-4 !m-0 opacity-70">{{
                          item.icon
                        }}</mat-icon>
                      }
                      <span class="font-bold text-[13px]">{{ item.name }}</span>
                    </div>
                    <span
                      class="text-[9px] font-mono opacity-40 bg-neutral-variant/10 px-1.5 py-0.5 rounded"
                    >
                      {{ item.id }}
                    </span>
                  </div>
                  @if (item.description) {
                    <span class="text-[10px] text-outline truncate opacity-70 mt-0.5">
                      {{ item.description }}
                    </span>
                  }
                </div>
              </mat-option>
            }
          </mat-optgroup>
        }
      </mat-autocomplete>

      @if (hint) {
        <mat-hint>{{ hint }}</mat-hint>
      }
    </mat-form-field>
  `,
})
export class DiscoverySuggestInputComponent implements OnInit, ControlValueAccessor {
  private discoveryService = inject(DiscoveryService);

  @Input() lookupCode = '';
  @Input() lookupLabel = '常用项建议';
  @Input() label = '输入值';
  @Input() placeholder = '输入自定义内容或 ${{ 引用变量 }}';
  @Input() hint = '';
  @Input() appearance: 'fill' | 'outline' = 'outline';
  @Input() subscriptSizing: 'fixed' | 'dynamic' = 'fixed';
  @Input() staticSuggestions: string[] = [];

  control = new FormControl('');
  items = signal<ModelsLookupItem[]>([]);
  isLoading = signal(false);
  disabled = false;

  private onChange: (value: any) => void = () => {};
  private onTouched: () => void = () => {};

  ngOnInit() {
    this.control.valueChanges
      .pipe(
        debounceTime(300),
        distinctUntilChanged(),
        switchMap((value) => {
          const search = typeof value === 'string' ? value : '';
          // Only trigger discovery search if code is provided and value doesn't look like a variable ref
          if (!this.lookupCode || search.includes('${{')) {
            return of({ items: [] });
          }

          this.isLoading.set(true);
          return this.discoveryService.discoveryLookupGet(this.lookupCode, search, 0, 10).pipe(
            catchError(() => of({ items: [] })),
            finalize(() => this.isLoading.set(false)),
          );
        }),
      )
      .subscribe((res) => {
        this.items.set(res.items || []);
      });

    // Notify parent on change
    this.control.valueChanges.subscribe((val) => this.onChange(val));
  }

  // ControlValueAccessor
  writeValue(value: any): void {
    this.control.setValue(value || '', { emitEvent: false });
  }
  registerOnChange(fn: any): void {
    this.onChange = fn;
  }
  registerOnTouched(fn: any): void {
    this.onTouched = fn;
  }
  setDisabledState(isDisabled: boolean): void {
    this.disabled = isDisabled;
    if (isDisabled) this.control.disable();
    else this.control.enable();
  }
}
