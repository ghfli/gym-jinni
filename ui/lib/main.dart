import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'package:provider/provider.dart';

import 'api_config.dart';
import 'auth_state.dart';

void main() {
  runApp(
    ChangeNotifierProvider(
      create: (_) => AuthState(),
      child: const GymJinniApp(),
    ),
  );
}

class GymJinniApp extends StatelessWidget {
  const GymJinniApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Gym Jinni',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(
          seedColor: Colors.teal,
          brightness: Brightness.light,
        ),
        useMaterial3: true,
        inputDecorationTheme: const InputDecorationTheme(
          border: OutlineInputBorder(),
          contentPadding: EdgeInsets.symmetric(horizontal: 16, vertical: 14),
        ),
      ),
      home: Consumer<AuthState>(
        builder: (_, auth, __) => auth.isLoggedIn ? const AppShell() : const AuthScreen(),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Navigation Shell
// ---------------------------------------------------------------------------

class AppShell extends StatefulWidget {
  const AppShell({super.key});

  @override
  State<AppShell> createState() => _AppShellState();
}

class _AppShellState extends State<AppShell> {
  int _idx = 0;

  void _switchTo(int idx) => setState(() => _idx = idx);

  static const _pages = <Widget>[
    DashboardPage(),
    ClassListPage(),
    MyBookingsPage(),
    ActivityPage(),
    ProfilePage(),
  ];

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: IndexedStack(index: _idx, children: _pages),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _idx,
        onDestinationSelected: (i) => setState(() => _idx = i),
        destinations: const [
          NavigationDestination(icon: Icon(Icons.home_outlined), selectedIcon: Icon(Icons.home), label: 'Home'),
          NavigationDestination(icon: Icon(Icons.fitness_center_outlined), selectedIcon: Icon(Icons.fitness_center), label: 'Classes'),
          NavigationDestination(icon: Icon(Icons.calendar_today_outlined), selectedIcon: Icon(Icons.calendar_today), label: 'Bookings'),
          NavigationDestination(icon: Icon(Icons.directions_run_outlined), selectedIcon: Icon(Icons.directions_run), label: 'Activity'),
          NavigationDestination(icon: Icon(Icons.person_outline), selectedIcon: Icon(Icons.person), label: 'Profile'),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Dashboard
// ---------------------------------------------------------------------------

class DashboardPage extends StatelessWidget {
  const DashboardPage({super.key});

  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthState>();
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(title: const Text('Gym Jinni')),
      body: ListView(
        padding: const EdgeInsets.all(20),
        children: [
          Text('Welcome, ${auth.userName ?? "Athlete"}!',
              style: theme.textTheme.headlineSmall),
          const SizedBox(height: 24),
          _SummaryCard(icon: Icons.fitness_center, label: 'Browse Classes', color: Colors.teal,
              onTap: () => _switchTab(context, 1)),
          const SizedBox(height: 12),
          _SummaryCard(icon: Icons.calendar_today, label: 'My Bookings', color: Colors.indigo,
              onTap: () => _switchTab(context, 2)),
          const SizedBox(height: 12),
          _SummaryCard(icon: Icons.directions_run, label: 'Activity', color: Colors.purple,
              onTap: () => _switchTab(context, 3)),
          const SizedBox(height: 12),
          _SummaryCard(icon: Icons.person, label: 'Profile', color: Colors.deepOrange,
              onTap: () => _switchTab(context, 4)),
        ],
      ),
    );
  }

  void _switchTab(BuildContext context, int idx) {
    final state = context.findAncestorStateOfType<_AppShellState>();
    state?._switchTo(idx);
  }
}

class _SummaryCard extends StatelessWidget {
  const _SummaryCard({required this.icon, required this.label, required this.color, required this.onTap});
  final IconData icon;
  final String label;
  final Color color;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
      color: color.withValues(alpha: 0.1),
      child: InkWell(
        borderRadius: BorderRadius.circular(16),
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 24),
          child: Row(children: [
            Icon(icon, size: 32, color: color),
            const SizedBox(width: 16),
            Text(label, style: Theme.of(context).textTheme.titleMedium?.copyWith(color: color)),
            const Spacer(),
            Icon(Icons.arrow_forward_ios, size: 16, color: color),
          ]),
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Class List & Detail
// ---------------------------------------------------------------------------

class ClassListPage extends StatefulWidget {
  const ClassListPage({super.key});

  @override
  State<ClassListPage> createState() => _ClassListPageState();
}

class _ClassListPageState extends State<ClassListPage> {
  List<dynamic> _classes = [];
  bool _loading = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    _fetchClasses();
  }

  Future<void> _fetchClasses() async {
    setState(() { _loading = true; _error = null; });
    final auth = context.read<AuthState>();
    try {
      final resp = await http.get(
        Uri.parse('${ApiConfig.baseUrl}/v1/classes'),
        headers: {'Accept': 'application/json', ...auth.authHeaders},
      );
      if (!mounted) return;
      if (resp.statusCode < 300) {
        final data = jsonDecode(resp.body) as Map<String, dynamic>;
        setState(() { _classes = (data['classes'] as List?) ?? []; _loading = false; });
      } else {
        setState(() { _error = 'Error ${resp.statusCode}: ${resp.body}'; _loading = false; });
      }
    } catch (e) {
      if (mounted) setState(() { _error = '$e'; _loading = false; });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Classes'), actions: [
        IconButton(onPressed: _fetchClasses, icon: const Icon(Icons.refresh)),
      ]),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : _error != null
              ? Center(child: Padding(padding: const EdgeInsets.all(20), child: Text(_error!, style: const TextStyle(color: Colors.red))))
              : _classes.isEmpty
                  ? const Center(child: Text('No classes available'))
                  : RefreshIndicator(
                      onRefresh: _fetchClasses,
                      child: ListView.builder(
                        padding: const EdgeInsets.all(12),
                        itemCount: _classes.length,
                        itemBuilder: (_, i) => _ClassTile(cls: _classes[i]),
                      ),
                    ),
    );
  }
}

class _ClassTile extends StatelessWidget {
  const _ClassTile({required this.cls});
  final dynamic cls;

  @override
  Widget build(BuildContext context) {
    final desc = cls['description'] ?? 'Untitled';
    final maxHd = cls['max_hdcnt'] ?? cls['maxHdcnt'];
    final subtitle = maxHd != null ? 'Max capacity: $maxHd' : '';

    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: ListTile(
        leading: const CircleAvatar(child: Icon(Icons.fitness_center)),
        title: Text(desc),
        subtitle: subtitle.isNotEmpty ? Text(subtitle) : null,
        trailing: FilledButton.tonal(
          onPressed: () => _bookClass(context),
          child: const Text('Book'),
        ),
      ),
    );
  }

  void _bookClass(BuildContext context) async {
    final auth = context.read<AuthState>();
    final classId = cls['id'];
    if (classId == null || auth.userId == null) return;

    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Confirm Booking'),
        content: Text('Book "${cls['description']}"?'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('Cancel')),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('Book')),
        ],
      ),
    );
    if (confirmed != true || !context.mounted) return;

    try {
      final resp = await http.post(
        Uri.parse('${ApiConfig.baseUrl}/v1/bookings'),
        headers: {'Content-Type': 'application/json', ...auth.authHeaders},
        body: jsonEncode({'user_id': auth.userId, 'class_id': classId}),
      );
      if (!context.mounted) return;
      final msg = resp.statusCode < 300 ? 'Booked successfully!' : 'Booking failed: ${resp.body}';
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(msg)));
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Error: $e')));
      }
    }
  }
}

// ---------------------------------------------------------------------------
// My Bookings
// ---------------------------------------------------------------------------

class MyBookingsPage extends StatefulWidget {
  const MyBookingsPage({super.key});

  @override
  State<MyBookingsPage> createState() => _MyBookingsPageState();
}

class _MyBookingsPageState extends State<MyBookingsPage> {
  List<dynamic> _bookings = [];
  bool _loading = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    _fetch();
  }

  Future<void> _fetch() async {
    setState(() { _loading = true; _error = null; });
    final auth = context.read<AuthState>();
    if (auth.userId == null) {
      setState(() { _error = 'Not logged in'; _loading = false; });
      return;
    }
    try {
      final resp = await http.get(
        Uri.parse('${ApiConfig.baseUrl}/v1/users/${auth.userId}/bookings'),
        headers: {'Accept': 'application/json', ...auth.authHeaders},
      );
      if (!mounted) return;
      if (resp.statusCode < 300) {
        final data = jsonDecode(resp.body) as Map<String, dynamic>;
        setState(() { _bookings = (data['bookings'] as List?) ?? []; _loading = false; });
      } else {
        setState(() { _error = 'Error ${resp.statusCode}'; _loading = false; });
      }
    } catch (e) {
      if (mounted) setState(() { _error = '$e'; _loading = false; });
    }
  }

  Future<void> _cancel(int bookingId) async {
    final auth = context.read<AuthState>();
    try {
      final resp = await http.post(
        Uri.parse('${ApiConfig.baseUrl}/v1/bookings/$bookingId/cancel'),
        headers: {'Content-Type': 'application/json', ...auth.authHeaders},
      );
      if (!mounted) return;
      if (resp.statusCode < 300) {
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('Booking cancelled')));
        _fetch();
      } else {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Cancel failed: ${resp.body}')));
      }
    } catch (e) {
      if (mounted) ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Error: $e')));
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('My Bookings'), actions: [
        IconButton(onPressed: _fetch, icon: const Icon(Icons.refresh)),
      ]),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : _error != null
              ? Center(child: Text(_error!))
              : _bookings.isEmpty
                  ? const Center(child: Text('No bookings yet'))
                  : RefreshIndicator(
                      onRefresh: _fetch,
                      child: ListView.builder(
                        padding: const EdgeInsets.all(12),
                        itemCount: _bookings.length,
                        itemBuilder: (_, i) {
                          final b = _bookings[i];
                          final status = b['status'] ?? 0;
                          final statusText = status == 1 ? 'Confirmed' : status == 2 ? 'Cancelled' : 'Unknown';
                          final color = status == 1 ? Colors.green : status == 2 ? Colors.red : Colors.grey;

                          return Card(
                            margin: const EdgeInsets.only(bottom: 8),
                            child: ListTile(
                              leading: CircleAvatar(backgroundColor: color.withValues(alpha: 0.15), child: Icon(Icons.event, color: color)),
                              title: Text('Class #${b['class_id'] ?? b['classId']}'),
                              subtitle: Text(statusText, style: TextStyle(color: color, fontWeight: FontWeight.w600)),
                              trailing: status == 1
                                  ? TextButton(
                                      onPressed: () => _cancel(b['id'] as int),
                                      child: const Text('Cancel', style: TextStyle(color: Colors.red)),
                                    )
                                  : null,
                            ),
                          );
                        },
                      ),
                    ),
    );
  }
}

// ---------------------------------------------------------------------------
// Activity
// ---------------------------------------------------------------------------

class ActivityPage extends StatefulWidget {
  const ActivityPage({super.key});

  @override
  State<ActivityPage> createState() => _ActivityPageState();
}

class _ActivityPageState extends State<ActivityPage> {
  List<dynamic> _activities = [];
  Map<String, dynamic>? _stats;
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _fetch();
  }

  Future<void> _fetch() async {
    setState(() => _loading = true);
    final auth = context.read<AuthState>();
    try {
      final results = await Future.wait([
        http.get(Uri.parse('${ApiConfig.baseUrl}/v1/users/${auth.userId}/activities'), headers: auth.authHeaders),
        http.get(Uri.parse('${ApiConfig.baseUrl}/v1/users/${auth.userId}/activity_stats'), headers: auth.authHeaders),
      ]);
      if (!mounted) return;
      if (results[0].statusCode < 300) {
        final data = jsonDecode(results[0].body) as Map<String, dynamic>;
        _activities = (data['activities'] as List?) ?? [];
      }
      if (results[1].statusCode < 300) {
        _stats = jsonDecode(results[1].body) as Map<String, dynamic>;
      }
    } catch (_) {}
    if (mounted) setState(() => _loading = false);
  }

  Future<void> _logActivity() async {
    final result = await showDialog<Map<String, dynamic>>(
      context: context,
      builder: (ctx) => const _LogActivityDialog(),
    );
    if (result == null || !mounted) return;

    final auth = context.read<AuthState>();
    try {
      final resp = await http.post(
        Uri.parse('${ApiConfig.baseUrl}/v1/activities'),
        headers: {'Content-Type': 'application/json', ...auth.authHeaders},
        body: jsonEncode({...result, 'user_id': auth.userId}),
      );
      if (!mounted) return;
      if (resp.statusCode < 300) {
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('Activity logged!')));
        _fetch();
      } else {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Error: ${resp.body}')));
      }
    } catch (e) {
      if (mounted) ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Error: $e')));
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Activity'), actions: [
        IconButton(onPressed: _fetch, icon: const Icon(Icons.refresh)),
      ]),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: _logActivity,
        icon: const Icon(Icons.add),
        label: const Text('Log'),
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : Column(children: [
              if (_stats != null)
                Padding(
                  padding: const EdgeInsets.all(16),
                  child: Row(children: [
                    _StatChip(label: 'Total', value: '${_stats!['total_count'] ?? 0}'),
                    const SizedBox(width: 8),
                    _StatChip(label: 'Minutes', value: '${_stats!['total_duration'] ?? 0}'),
                    const SizedBox(width: 8),
                    _StatChip(label: 'Calories', value: '${_stats!['total_calories'] ?? 0}'),
                  ]),
                ),
              Expanded(
                child: _activities.isEmpty
                    ? const Center(child: Text('No activities yet'))
                    : ListView.builder(
                        padding: const EdgeInsets.symmetric(horizontal: 12),
                        itemCount: _activities.length,
                        itemBuilder: (_, i) {
                          final a = _activities[i];
                          return Card(
                            margin: const EdgeInsets.only(bottom: 8),
                            child: ListTile(
                              leading: CircleAvatar(child: Icon(_activityIcon(a['activity_type'] ?? ''))),
                              title: Text(a['title'] ?? 'Activity'),
                              subtitle: Text('${a['duration_minutes'] ?? 0} min | ${a['calories'] ?? 0} cal'),
                            ),
                          );
                        },
                      ),
              ),
            ]),
    );
  }

  IconData _activityIcon(String type) {
    switch (type) {
      case 'workout': return Icons.fitness_center;
      case 'run': return Icons.directions_run;
      case 'class': return Icons.group;
      default: return Icons.sports;
    }
  }
}

class _StatChip extends StatelessWidget {
  const _StatChip({required this.label, required this.value});
  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return Expanded(
      child: Card(
        child: Padding(
          padding: const EdgeInsets.symmetric(vertical: 12, horizontal: 8),
          child: Column(children: [
            Text(value, style: Theme.of(context).textTheme.headlineSmall),
            Text(label, style: Theme.of(context).textTheme.bodySmall),
          ]),
        ),
      ),
    );
  }
}

class _LogActivityDialog extends StatefulWidget {
  const _LogActivityDialog();

  @override
  State<_LogActivityDialog> createState() => _LogActivityDialogState();
}

class _LogActivityDialogState extends State<_LogActivityDialog> {
  final _title = TextEditingController();
  final _duration = TextEditingController();
  final _calories = TextEditingController();
  String _type = 'workout';

  @override
  void dispose() {
    _title.dispose();
    _duration.dispose();
    _calories.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('Log Activity'),
      content: SingleChildScrollView(
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          DropdownButtonFormField<String>(
            initialValue: _type,
            decoration: const InputDecoration(labelText: 'Type'),
            items: const [
              DropdownMenuItem(value: 'workout', child: Text('Workout')),
              DropdownMenuItem(value: 'run', child: Text('Run')),
              DropdownMenuItem(value: 'class', child: Text('Class')),
            ],
            onChanged: (v) => setState(() => _type = v ?? 'workout'),
          ),
          const SizedBox(height: 8),
          TextField(controller: _title, decoration: const InputDecoration(labelText: 'Title')),
          const SizedBox(height: 8),
          TextField(controller: _duration, decoration: const InputDecoration(labelText: 'Duration (min)'), keyboardType: TextInputType.number),
          const SizedBox(height: 8),
          TextField(controller: _calories, decoration: const InputDecoration(labelText: 'Calories'), keyboardType: TextInputType.number),
        ]),
      ),
      actions: [
        TextButton(onPressed: () => Navigator.pop(context), child: const Text('Cancel')),
        FilledButton(
          onPressed: () => Navigator.pop(context, {
            'activity_type': _type,
            'title': _title.text,
            'duration_minutes': int.tryParse(_duration.text) ?? 0,
            'calories': int.tryParse(_calories.text) ?? 0,
          }),
          child: const Text('Save'),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Profile
// ---------------------------------------------------------------------------

class ProfilePage extends StatelessWidget {
  const ProfilePage({super.key});

  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthState>();
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(title: const Text('Profile')),
      body: ListView(
        padding: const EdgeInsets.all(20),
        children: [
          CircleAvatar(
            radius: 48,
            backgroundColor: theme.colorScheme.primaryContainer,
            child: Text(
              (auth.userName ?? '?')[0].toUpperCase(),
              style: theme.textTheme.headlineLarge?.copyWith(color: theme.colorScheme.onPrimaryContainer),
            ),
          ),
          const SizedBox(height: 16),
          Text(auth.userName ?? 'Unknown', textAlign: TextAlign.center, style: theme.textTheme.titleLarge),
          Text('User #${auth.userId ?? "?"}', textAlign: TextAlign.center, style: theme.textTheme.bodyMedium?.copyWith(color: Colors.grey)),
          const SizedBox(height: 32),
          OutlinedButton.icon(
            onPressed: () => auth.logout(),
            icon: const Icon(Icons.logout),
            label: const Text('Log out'),
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Auth Screen (Register + Login)
// ---------------------------------------------------------------------------

class AuthScreen extends StatefulWidget {
  const AuthScreen({super.key});

  @override
  State<AuthScreen> createState() => _AuthScreenState();
}

class _AuthScreenState extends State<AuthScreen> with SingleTickerProviderStateMixin {
  late final TabController _tabController;

  final _regEmail = TextEditingController();
  final _regPhone = TextEditingController();
  final _regName = TextEditingController();
  final _regPass = TextEditingController();

  final _loginEmail = TextEditingController();
  final _loginPass = TextEditingController();

  String? _message;
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 2, vsync: this);
  }

  @override
  void dispose() {
    _tabController.dispose();
    _regEmail.dispose();
    _regPhone.dispose();
    _regName.dispose();
    _regPass.dispose();
    _loginEmail.dispose();
    _loginPass.dispose();
    super.dispose();
  }

  Future<void> _register() async {
    setState(() { _busy = true; _message = null; });
    try {
      final resp = await http.post(
        Uri.parse('${ApiConfig.baseUrl}/v1/create_user'),
        headers: {'Content-Type': 'application/json', 'Accept': 'application/json'},
        body: jsonEncode({
          'user': {
            'email': _regEmail.text.trim(),
            'phone': _regPhone.text.trim(),
            'name': _regName.text.trim(),
            'passwd': _regPass.text,
          },
        }),
      );
      if (!mounted) return;
      if (resp.statusCode < 300) {
        final map = jsonDecode(resp.body) as Map<String, dynamic>;
        final user = map['user'] as Map<String, dynamic>?;
        setState(() => _message = 'Created user id: ${user?['id']}. Switch to Log in.');
        _tabController.animateTo(1);
      } else {
        setState(() => _message = 'Error ${resp.statusCode}: ${resp.body}');
      }
    } catch (e) {
      if (mounted) setState(() => _message = 'Request failed: $e');
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _login() async {
    setState(() { _busy = true; _message = null; });
    try {
      final resp = await http.post(
        Uri.parse('${ApiConfig.baseUrl}/v1/login_user'),
        headers: {'Content-Type': 'application/json', 'Accept': 'application/json'},
        body: jsonEncode({
          'user': {
            'email': _loginEmail.text.trim(),
            'phone': '123456789',
            'name': 'user',
            'passwd': _loginPass.text,
          },
        }),
      );
      if (!mounted) return;
      if (resp.statusCode < 300) {
        final map = jsonDecode(resp.body) as Map<String, dynamic>;
        final auth = context.read<AuthState>();
        auth.login(
          accessToken: (map['access_tkn'] ?? map['accessTkn'] ?? '') as String,
          refreshToken: (map['refresh_tkn'] ?? map['refreshTkn'] ?? '') as String,
          userId: (map['user']?['id'] ?? 0) as int,
          userName: (map['user']?['name'] ?? '') as String,
        );
      } else {
        setState(() => _message = 'Error ${resp.statusCode}: ${resp.body}');
      }
    } catch (e) {
      if (mounted) setState(() => _message = 'Request failed: $e');
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 420),
            child: Column(
              children: [
                const SizedBox(height: 40),
                Icon(Icons.fitness_center, size: 56, color: Theme.of(context).colorScheme.primary),
                const SizedBox(height: 8),
                Text('Gym Jinni', style: Theme.of(context).textTheme.headlineMedium?.copyWith(fontWeight: FontWeight.bold)),
                const SizedBox(height: 24),
                TabBar(controller: _tabController, tabs: const [Tab(text: 'Register'), Tab(text: 'Log in')]),
                if (_message != null)
                  Padding(
                    padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
                    child: Text(_message!, style: TextStyle(color: Theme.of(context).colorScheme.error)),
                  ),
                Expanded(
                  child: TabBarView(controller: _tabController, children: [
                    _buildRegisterForm(),
                    _buildLoginForm(),
                  ]),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildRegisterForm() {
    return ListView(padding: const EdgeInsets.all(20), children: [
      TextField(controller: _regEmail, decoration: const InputDecoration(labelText: 'Email', prefixIcon: Icon(Icons.email_outlined)),
          keyboardType: TextInputType.emailAddress),
      const SizedBox(height: 12),
      TextField(controller: _regPhone, decoration: const InputDecoration(labelText: 'Phone', prefixIcon: Icon(Icons.phone_outlined)),
          keyboardType: TextInputType.phone),
      const SizedBox(height: 12),
      TextField(controller: _regName, decoration: const InputDecoration(labelText: 'Name', prefixIcon: Icon(Icons.person_outline)),
          textCapitalization: TextCapitalization.words),
      const SizedBox(height: 12),
      TextField(controller: _regPass, decoration: const InputDecoration(labelText: 'Password', prefixIcon: Icon(Icons.lock_outline)),
          obscureText: true),
      const SizedBox(height: 20),
      FilledButton(onPressed: _busy ? null : _register,
          child: _busy ? const SizedBox(width: 22, height: 22, child: CircularProgressIndicator(strokeWidth: 2)) : const Text('Create account')),
    ]);
  }

  Widget _buildLoginForm() {
    return ListView(padding: const EdgeInsets.all(20), children: [
      TextField(controller: _loginEmail, decoration: const InputDecoration(labelText: 'Email', prefixIcon: Icon(Icons.email_outlined)),
          keyboardType: TextInputType.emailAddress),
      const SizedBox(height: 12),
      TextField(controller: _loginPass, decoration: const InputDecoration(labelText: 'Password', prefixIcon: Icon(Icons.lock_outline)),
          obscureText: true),
      const SizedBox(height: 20),
      FilledButton(onPressed: _busy ? null : _login,
          child: _busy ? const SizedBox(width: 22, height: 22, child: CircularProgressIndicator(strokeWidth: 2)) : const Text('Log in')),
    ]);
  }
}
