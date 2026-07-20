from odoo import models, fields

class BenchmarkArticle(models.Model):
    _name = 'benchmark.article'
    _description = 'Benchmark Article'

    title = fields.Char(string='Title', required=True)
    content = fields.Text(string='Content')
