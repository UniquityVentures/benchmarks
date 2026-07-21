from odoo import models, fields

class BenchmarkTask(models.Model):
    _name = 'benchmark.task'
    _description = 'Benchmark Task'

    val = fields.Integer(string='Val')
    result = fields.Integer(string='Result')
    state = fields.Selection([
        ('pending', 'Pending'),
        ('completed', 'Completed')
    ], default='pending')

    def run_increment_job(self):
        self.ensure_one()
        self.write({
            'result': self.val + 1,
            'state': 'completed'
        })
