package selectedv2data

import "ocr_soul/internal/trainer"

// Samples are transcribed from the visible soul title, Chinese slot label,
// and attribute rows in train/assets/selected-v2. The blank, cleared-panel
// screenshot is intentionally omitted.
var Samples = []trainer.SelectedSoul{
	{File: "MuMu-20260924-184447-961.png", Type: "海月火玉", Position: 4, Level: -1, Main: trainer.SelectedAttribute{Name: "生命加成", Value: "10.00%"}, Subs: []trainer.SelectedAttribute{{Name: "防御", Value: "4.57"}, {Name: "暴击伤害", Value: "3.65%"}}},
	{File: "MuMu-20260924-184807-341.png", Type: "油赤子", Position: 2, Level: -1, Main: trainer.SelectedAttribute{Name: "防御加成", Value: "10.00%"}, Subs: []trainer.SelectedAttribute{{Name: "生命", Value: "95.09"}, {Name: "效果抵抗", Value: "3.34%"}}},
	{File: "MuMu-20260924-184825-024.png", Type: "油赤子", Position: 2, Level: -1, Main: trainer.SelectedAttribute{Name: "防御加成", Value: "10.00%"}, Subs: []trainer.SelectedAttribute{{Name: "生命", Value: "95.09"}, {Name: "效果抵抗", Value: "3.34%"}}},
	{File: "MuMu-20260924-184900-178.png", Type: "油赤子", Position: 2, Level: -1, Main: trainer.SelectedAttribute{Name: "防御加成", Value: "10.00%"}, Subs: []trainer.SelectedAttribute{{Name: "生命", Value: "95.09"}, {Name: "效果抵抗", Value: "3.34%"}}},
	{File: "MuMu-20260924-191239-905.png", Type: "海月火玉", Position: 4, Level: -1, Main: trainer.SelectedAttribute{Name: "生命加成", Value: "10.00%"}, Subs: []trainer.SelectedAttribute{{Name: "防御", Value: "4.57"}, {Name: "暴击伤害", Value: "3.65%"}}},
	{File: "MuMu-20260924-191242-925.png", Type: "海月火玉", Position: 1, Level: -1, Main: trainer.SelectedAttribute{Name: "攻击", Value: "81.00"}, Subs: []trainer.SelectedAttribute{{Name: "防御", Value: "4.32"}, {Name: "生命加成", Value: "2.90%"}, {Name: "暴击伤害", Value: "3.67%"}}},
	{File: "MuMu-20260924-191245-069.png", Type: "片叶之苇", Position: 3, Level: -1, Main: trainer.SelectedAttribute{Name: "防御", Value: "14.00"}, Subs: []trainer.SelectedAttribute{{Name: "生命加成", Value: "2.79%"}, {Name: "速度", Value: "2.42"}, {Name: "暴击", Value: "2.90%"}}},
	{File: "MuMu-20260924-191249-641.png", Type: "油赤子", Position: 4, Level: -1, Main: trainer.SelectedAttribute{Name: "攻击加成", Value: "10.00%"}, Subs: []trainer.SelectedAttribute{{Name: "攻击加成", Value: "2.67%"}, {Name: "速度", Value: "2.61"}, {Name: "暴击伤害", Value: "3.66%"}}},
	{File: "MuMu-20260924-191251-962.png", Type: "隐念", Position: 6, Level: -1, Main: trainer.SelectedAttribute{Name: "防御加成", Value: "10.00%"}, Subs: []trainer.SelectedAttribute{{Name: "生命", Value: "98.22"}, {Name: "攻击", Value: "24.55"}, {Name: "暴击", Value: "2.94%"}}},
	{File: "MuMu-20260924-191257-426.png", Type: "片叶之苇", Position: 6, Level: -1, Main: trainer.SelectedAttribute{Name: "防御加成", Value: "10.00%"}, Subs: []trainer.SelectedAttribute{{Name: "生命", Value: "111.40"}, {Name: "速度", Value: "2.95"}, {Name: "暴击伤害", Value: "3.97%"}}},
	{File: "MuMu-20260924-191300-401.png", Type: "招财猫", Position: 1, Level: -1, Main: trainer.SelectedAttribute{Name: "攻击", Value: "81.00"}, Subs: []trainer.SelectedAttribute{{Name: "防御加成", Value: "2.64%"}, {Name: "速度", Value: "2.50"}, {Name: "暴击伤害", Value: "3.77%"}}},
	{File: "MuMu-20260924-191302-809.png", Type: "海月火玉", Position: 6, Level: -1, Main: trainer.SelectedAttribute{Name: "攻击加成", Value: "10.00%"}, Subs: []trainer.SelectedAttribute{{Name: "防御", Value: "4.70"}, {Name: "生命加成", Value: "2.90%"}, {Name: "攻击加成", Value: "2.85%"}, {Name: "暴击伤害", Value: "3.34%"}}},
	{File: "MuMu-20260924-191305-280.png", Type: "片叶之苇", Position: 1, Level: -1, Main: trainer.SelectedAttribute{Name: "攻击", Value: "81.00"}, Subs: []trainer.SelectedAttribute{{Name: "防御加成", Value: "2.69%"}, {Name: "暴击伤害", Value: "3.66%"}}},
	{File: "MuMu-20260924-191310-921.png", Type: "钓瓶火", Position: 4, Level: -1, Main: trainer.SelectedAttribute{Name: "攻击加成", Value: "10.00%"}, Subs: []trainer.SelectedAttribute{{Name: "防御", Value: "4.12"}, {Name: "防御加成", Value: "3.00%"}, {Name: "暴击伤害", Value: "3.53%"}}},
	{File: "MuMu-20260924-191313-833.png", Type: "魍魉之匣", Position: 6, Level: -1, Main: trainer.SelectedAttribute{Name: "生命加成", Value: "10.00%"}, Subs: []trainer.SelectedAttribute{{Name: "攻击加成", Value: "2.88%"}, {Name: "暴击伤害", Value: "3.45%"}, {Name: "效果命中", Value: "3.22%"}}},
	{File: "MuMu-20260924-191316-961.png", Type: "土蜘蛛", Position: 1, Level: -1, Main: trainer.SelectedAttribute{Name: "攻击", Value: "81.00"}, Subs: []trainer.SelectedAttribute{{Name: "攻击", Value: "23.66"}, {Name: "生命加成", Value: "2.45%"}, {Name: "暴击", Value: "2.83%"}, {Name: "效果命中", Value: "3.20%"}}},
	{File: "MuMu-20260924-191319-154.png", Type: "轮入道", Position: 2, Level: -1, Main: trainer.SelectedAttribute{Name: "防御加成", Value: "10.00%"}, Subs: []trainer.SelectedAttribute{{Name: "生命", Value: "94.00"}, {Name: "防御加成", Value: "2.81%"}, {Name: "效果抵抗", Value: "3.45%"}, {Name: "暴击伤害", Value: "3.90%"}}},
	{File: "MuMu-20260924-191321-290.png", Type: "木魅", Position: 5, Level: -1, Main: trainer.SelectedAttribute{Name: "生命", Value: "342.00"}, Subs: []trainer.SelectedAttribute{{Name: "攻击", Value: "24.87"}, {Name: "暴击伤害", Value: "3.74%"}}},
	{File: "MuMu-20260924-191326-218.png", Type: "兵主部", Position: 5, Level: -1, Main: trainer.SelectedAttribute{Name: "生命", Value: "342.00"}, Subs: []trainer.SelectedAttribute{{Name: "攻击", Value: "21.86"}, {Name: "生命加成", Value: "2.71%"}, {Name: "暴击伤害", Value: "3.31%"}, {Name: "效果抵抗", Value: "3.48%"}}},
	{File: "MuMu-20260924-191430-862.png", Type: "土蜘蛛", Position: 1, Level: -1, Main: trainer.SelectedAttribute{Name: "攻击", Value: "81.00"}, Subs: []trainer.SelectedAttribute{{Name: "攻击", Value: "23.66"}, {Name: "生命加成", Value: "2.45%"}, {Name: "暴击", Value: "2.83%"}, {Name: "效果命中", Value: "3.20%"}}},
}
