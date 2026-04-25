import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;

import 'api_config.dart';

void main() {
  runApp(const GymJinniApp());
}

class GymJinniApp extends StatelessWidget {
  const GymJinniApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Gym Jinni',
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.teal),
        useMaterial3: true,
      ),
      home: const AuthScreen(),
    );
  }
}

class AuthScreen extends StatefulWidget {
  const AuthScreen({super.key});

  @override
  State<AuthScreen> createState() => _AuthScreenState();
}

class _AuthScreenState extends State<AuthScreen>
    with SingleTickerProviderStateMixin {
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
    setState(() {
      _busy = true;
      _message = null;
    });
    final uri = Uri.parse('${ApiConfig.baseUrl}/v1/create_user');
    final body = jsonEncode({
      'user': {
        'email': _regEmail.text.trim(),
        'phone': _regPhone.text.trim(),
        'name': _regName.text.trim(),
        'passwd': _regPass.text,
      },
    });
    try {
      final resp = await http.post(
        uri,
        headers: {'Content-Type': 'application/json', 'Accept': 'application/json'},
        body: body,
      );
      if (!mounted) return;
      if (resp.statusCode >= 200 && resp.statusCode < 300) {
        final map = jsonDecode(resp.body) as Map<String, dynamic>;
        final user = map['user'] as Map<String, dynamic>?;
        final id = user?['id'];
        setState(() => _message = 'Created user id: $id');
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
    setState(() {
      _busy = true;
      _message = null;
    });
    final uri = Uri.parse('${ApiConfig.baseUrl}/v1/login_user');
    // API validators require a full [User] message; only email + password are used server-side.
    const placeholderPhone = '123456789';
    const placeholderName = 'user';
    final body = jsonEncode({
      'user': {
        'email': _loginEmail.text.trim(),
        'phone': placeholderPhone,
        'name': placeholderName,
        'passwd': _loginPass.text,
      },
    });
    try {
      final resp = await http.post(
        uri,
        headers: {'Content-Type': 'application/json', 'Accept': 'application/json'},
        body: body,
      );
      if (!mounted) return;
      if (resp.statusCode >= 200 && resp.statusCode < 300) {
        final map = jsonDecode(resp.body) as Map<String, dynamic>;
        final sid = map['session_id'] ?? map['sessionId'];
        setState(() => _message = 'Logged in (session: $sid)');
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
      appBar: AppBar(
        title: const Text('Gym Jinni'),
        bottom: TabBar(
          controller: _tabController,
          tabs: const [
            Tab(text: 'Register'),
            Tab(text: 'Log in'),
          ],
        ),
      ),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(8),
            child: Text(
              'API: ${ApiConfig.baseUrl}',
              style: Theme.of(context).textTheme.labelSmall,
            ),
          ),
          if (_message != null)
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16),
              child: SelectableText(_message!, key: const Key('status_message')),
            ),
          Expanded(
            child: TabBarView(
              controller: _tabController,
              children: [
                _RegisterForm(
                  email: _regEmail,
                  phone: _regPhone,
                  name: _regName,
                  password: _regPass,
                  busy: _busy,
                  onSubmit: _register,
                ),
                _LoginForm(
                  email: _loginEmail,
                  password: _loginPass,
                  busy: _busy,
                  onSubmit: _login,
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _RegisterForm extends StatelessWidget {
  const _RegisterForm({
    required this.email,
    required this.phone,
    required this.name,
    required this.password,
    required this.busy,
    required this.onSubmit,
  });

  final TextEditingController email;
  final TextEditingController phone;
  final TextEditingController name;
  final TextEditingController password;
  final bool busy;
  final VoidCallback onSubmit;

  @override
  Widget build(BuildContext context) {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        TextField(
          controller: email,
          decoration: const InputDecoration(labelText: 'Email'),
          keyboardType: TextInputType.emailAddress,
          autofillHints: const [AutofillHints.email],
        ),
        TextField(
          controller: phone,
          decoration: const InputDecoration(labelText: 'Phone (3–15 digits)'),
          keyboardType: TextInputType.phone,
        ),
        TextField(
          controller: name,
          decoration: const InputDecoration(labelText: 'Name'),
          textCapitalization: TextCapitalization.words,
        ),
        TextField(
          controller: password,
          decoration: const InputDecoration(labelText: 'Password'),
          obscureText: true,
        ),
        const SizedBox(height: 16),
        FilledButton(
          onPressed: busy ? null : onSubmit,
          child: busy
              ? const SizedBox(
                  width: 22,
                  height: 22,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : const Text('Create account'),
        ),
      ],
    );
  }
}

class _LoginForm extends StatelessWidget {
  const _LoginForm({
    required this.email,
    required this.password,
    required this.busy,
    required this.onSubmit,
  });

  final TextEditingController email;
  final TextEditingController password;
  final bool busy;
  final VoidCallback onSubmit;

  @override
  Widget build(BuildContext context) {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        TextField(
          controller: email,
          decoration: const InputDecoration(labelText: 'Email'),
          keyboardType: TextInputType.emailAddress,
          autofillHints: const [AutofillHints.email],
        ),
        TextField(
          controller: password,
          decoration: const InputDecoration(labelText: 'Password'),
          obscureText: true,
        ),
        const SizedBox(height: 16),
        FilledButton(
          onPressed: busy ? null : onSubmit,
          child: busy
              ? const SizedBox(
                  width: 22,
                  height: 22,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : const Text('Log in'),
        ),
      ],
    );
  }
}
